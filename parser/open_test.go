package parser

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/Velocidex/ordereddict"
)

// A compact journal holding two DATA objects, an ENTRY_ARRAY, one ENTRY with two items, and an
// xz flagged DATA object written directly after the ENTRY. Walking 16 bytes past the ENTRY reads
// that DATA's type and flags (01 01) as an item offset of 257, which lands in the file header.
func compactJournalWithXZDataAfterEntry() []byte {
	const (
		headerSize = 272
		dataA      = 272 // MESSAGE=hello
		dataB      = 360 // PRIORITY=6
		entryArray = 448 // 0x1c0, the 01 at byte 257 of the header
		entry      = 480
		xzData     = 552
		fileEnd    = 4096 // GetLogs stops at arena_size, not header_size+arena_size
	)

	b := make([]byte, fileEnd)
	le := binary.LittleEndian

	copy(b, "LPKSHHRH")
	le.PutUint32(b[12:], 1|16) // COMPRESSED_XZ | COMPACT
	le.PutUint64(b[88:], headerSize)
	le.PutUint64(b[96:], fileEnd-headerSize)
	le.PutUint64(b[144:], 5) // n_objects
	le.PutUint64(b[152:], 1) // n_entries
	le.PutUint32(b[256:], entryArray)
	le.PutUint64(b[264:], entry)

	object := func(offset int, objectType, flags byte, size int) {
		b[offset], b[offset+1] = objectType, flags
		le.PutUint64(b[offset+8:], uint64(size))
	}
	data := func(offset int, flags byte, payload string) {
		object(offset, 1, flags, 16+56+len(payload))
		copy(b[offset+16+56:], payload)
	}

	data(dataA, 0, "MESSAGE=hello")
	data(dataB, 0, "PRIORITY=6")

	object(entryArray, 6, 0, 16+8+4)
	le.PutUint32(b[entryArray+16+8:], entry)

	object(entry, 3, 0, 16+48+2*4)
	le.PutUint64(b[entry+16:], 1) // seqnum
	le.PutUint32(b[entry+16+48:], dataA)
	le.PutUint32(b[entry+16+48+4:], dataB)

	data(xzData, 1, "\xfd7zXZ\x00\x00\x00")

	return b
}

func TestGetLogsStopsAtEndOfEntryObject(t *testing.T) {
	rows := readLogs(t, bytes.NewReader(compactJournalWithXZDataAfterEntry()))

	if len(rows) != 1 {
		t.Fatalf("got %d entries, want 1", len(rows))
	}
	fields := eventData(rows[0])
	if got := fmt.Sprint(fields.Keys()); got != "[MESSAGE PRIORITY]" {
		t.Errorf("got fields %s, want [MESSAGE PRIORITY]", got)
	}
	if got, _ := fields.GetString("MESSAGE"); got != "hello" {
		t.Errorf("got MESSAGE %q, want hello", got)
	}
}

// testdata/compressed_after_entry.journal.gz was written by systemd-journal-remote (systemd 255)
// and passes `journalctl --verify`. Its entries have an even item count and are each followed by
// a zstd DATA object, so the same walk past the ENTRY reads an item offset of 1025, inside the
// field hash table.
func TestGetLogsSystemdJournalWithCompressedDataAfterEntry(t *testing.T) {
	fd, err := os.Open("testdata/compressed_after_entry.journal.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer fd.Close()

	gz, err := gzip.NewReader(fd)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}

	rows := readLogs(t, bytes.NewReader(journal))

	if len(rows) != 58 {
		t.Fatalf("got %d entries, want 58", len(rows))
	}
	if got, _ := eventData(rows[57]).GetString("SYSLOG_IDENTIFIER"); got != "kernel" {
		t.Errorf("got last entry from %q, want the kernel boot line", got)
	}
}

func readLogs(t *testing.T, reader io.ReaderAt) []*ordereddict.Dict {
	t.Helper()

	journal, err := OpenFile(reader)
	if err != nil {
		t.Fatal(err)
	}

	var rows []*ordereddict.Dict
	for row := range journal.GetLogs(context.Background()) {
		rows = append(rows, row)
	}
	return rows
}

func eventData(row *ordereddict.Dict) *ordereddict.Dict {
	value, _ := row.Get("EventData")
	return value.(*ordereddict.Dict)
}

// A crash-consistent disk snapshot can catch journald mid-append: the ENTRY is written but a DATA
// object after it still reads size 0, which made the payload length negative and panicked.
func TestGetLogsSkipsDataObjectWithCorruptSize(t *testing.T) {
	journal := compactJournalWithXZDataAfterEntry()
	le := binary.LittleEndian
	le.PutUint32(journal[480+16+48+4:], 552) // the ENTRY's second item -> the DATA object after it
	journal[552+1] = 0                       // uncompressed
	le.PutUint64(journal[552+8:], 0)         // and its size never written

	rows := readLogs(t, bytes.NewReader(journal))

	if len(rows) != 1 {
		t.Fatalf("got %d entries, want 1", len(rows))
	}
	if got, _ := eventData(rows[0]).GetString("MESSAGE"); got != "hello" {
		t.Errorf("got MESSAGE %q, want hello", got)
	}
}
