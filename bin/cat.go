package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Velocidex/go-journalctl/parser"
	kingpin "github.com/alecthomas/kingpin/v2"
	ntfs_parser "www.velocidex.com/golang/go-ntfs/parser"
)

var (
	cat_command = app.Command(
		"cat", "Dump all events from file.")

	cat_command_file_arg = cat_command.Arg(
		"file", "The journal file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))

	cat_command_raw = cat_command.Flag(
		"raw", "Emit raw events instead",
	).Short('r').Bool()

	cat_command_follow = cat_command.Flag(
		"follow", "Follow the file and emit additional entried.",
	).Short('f').Bool()
)

func doCat() {
	reader, _ := ntfs_parser.NewPagedReader(
		getReader(*cat_command_file_arg), 1024, 10000)

	journal, err := parser.OpenFile(reader)
	kingpin.FatalIfError(err, "Can not open filesystem")

	if *cat_command_raw {
		journal.RawLogs = true
	}

	if *cat_command_follow {
		// Only print newer events from now on.
		journal.MinSeq = journal.GetLastSequence()

		for {
			last_seq := journal.GetLastSequence()
			if journal.MinSeq != last_seq {
				PrintOnce(journal)
				journal.MinSeq = last_seq
			}
			time.Sleep(time.Second)
			reader.Flush()
		}
	} else {
		PrintOnce(journal)
	}
}

func PrintOnce(journal *parser.JournalFile) {
	for log := range journal.GetLogs() {
		fmt.Printf("%v\n", log)
	}
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case "cat":
			doCat()
		default:
			return false
		}
		return true
	})
}
