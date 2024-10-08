package parser

import (
	"errors"
	"io"
	"time"

	"github.com/Velocidex/ordereddict"
)

type JournalFile struct {
	Reader  io.ReaderAt
	Profile *JournalProfile
	Header  *FileHeader

	IsCompact  bool
	NumEntries uint64
	NumObjects uint64
	ArenaSize  int64

	// The minimum sequence number to process
	MinSeq  uint64
	MinTime time.Time
	MaxTime time.Time

	RawLogs bool
}

func (self *JournalFile) DebugString() string {
	return self.Header.DebugString()
}

func (self *JournalFile) GetLastSequence() uint64 {
	return self.Header.tail_entry_seqnum()
}

func (self *JournalFile) GetLogs() chan *ordereddict.Dict {
	output_chan := make(chan *ordereddict.Dict)

	var time_start, time_end int64

	if !self.MinTime.IsZero() {
		time_start = self.MinTime.UnixNano() / 1000
	}

	if !self.MaxTime.IsZero() {
		time_end = self.MaxTime.UnixNano() / 1000
	} else {
		time_end = int64(^uint64(0) >> 1)
	}

	go func() {
		defer close(output_chan)

		i := self.Header.header_size()
		for i <= self.Header.arena_size() {
			obj := self.Profile.ObjectHeader(self.Reader, i)
			obj_size := obj.__real_size()

			switch obj.Type().Name {
			case "OBJECT_ENTRY":
				// OBJECT_ENTRY follows the object header
				entry := self.Profile.EntryObject(
					self.Reader, i+int64(obj.Size()))

				rt := entry.realtime()

				// Check the entry time is within range.
				if rt >= time_start && rt <= time_end &&

					// Only forward entries with sequence number
					// higher than the minimum required
					entry.seqnum() > self.MinSeq {

					var row *ordereddict.Dict
					if self.RawLogs {
						row = entry.GetRaw(self, obj_size)
					} else {
						row = entry.GetParsed(self, obj_size)
					}

					output_chan <- row
				}
			}

			// Size is rounded up to be 64 bit aligned
			if obj_size%8 > 0 {
				obj_size += 8 - obj_size%8
			}
			i += obj_size

			if obj_size <= 0 {
				break
			}
		}
	}()

	return output_chan
}

func OpenFile(reader io.ReaderAt) (*JournalFile, error) {
	self := &JournalFile{
		Reader:  reader,
		Profile: NewJournalProfile(),
	}

	self.Header = self.Profile.FileHeader(reader, 0)
	if self.Header.Signature() != "LPKSHHRH" {
		return nil, errors.New("Unknown signature!")
	}

	self.IsCompact = self.Header.incompatible_flags().IsSet("COMPACT")
	self.NumEntries = self.Header.n_entries()
	self.NumObjects = self.Header.n_objects()
	self.ArenaSize = self.Header.arena_size()

	return self, nil
}
