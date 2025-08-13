package main

import (
	"fmt"
	"io"
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
	).Required().String()

	cat_command_raw = cat_command.Flag(
		"raw", "Emit raw events instead",
	).Short('r').Bool()

	cat_command_follow = cat_command.Flag(
		"follow", "Follow the file and emit additional entried.",
	).Short('f').Bool()

	cat_command_start = cat_command.Flag(
		"start", "Start time in RFC3339 format eg 2014-11-12T11:45:26.371Z").String()

	cat_command_end = cat_command.Flag(
		"end", "End time in RFC3339 format eg 2014-11-12T11:45:26.371Z ").String()
)

func doCat() {
	ctx, cancel := install_sig_handler()
	defer cancel()

	if *cat_command_follow {
		for event := range parser.WatchFile(
			ctx, func() (io.ReaderAt, func(), error) {
				// Open the file fresh each iteration in case the file
				// was rotated. See
				// https://github.com/Velocidex/go-journalctl/issues/4
				fd, err := os.Open(*cat_command_file_arg)
				if err != nil {
					return nil, nil, err
				}
				reader, err := ntfs_parser.NewPagedReader(
					fd, 1024, 10000)

				return reader, func() {
					fd.Close()
				}, err
			}) {
			fmt.Printf("%v\n", event)
		}

		return
	}

	fd, err := os.Open(*cat_command_file_arg)
	kingpin.FatalIfError(err, "Can not open file")

	reader, _ := ntfs_parser.NewPagedReader(fd, 1024, 10000)

	journal, err := parser.OpenFile(reader)
	kingpin.FatalIfError(err, "Can not open filesystem")

	if *cat_command_raw {
		journal.RawLogs = true
	}

	if *cat_command_start != "" {
		journal.MinTime, err = time.Parse(time.RFC3339, *cat_command_start)
		kingpin.FatalIfError(err, "Can not parse start time, use RFC3339 format, eg 2014-11-12T11:45:26.371Z")
	}

	if *cat_command_end != "" {
		journal.MaxTime, err = time.Parse(time.RFC3339, *cat_command_end)
		kingpin.FatalIfError(err, "Can not parse end time, use RFC3339 format, eg 2014-11-12T11:45:26.371Z")
	}

	PrintOnce(journal)
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
