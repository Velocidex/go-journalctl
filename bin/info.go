package main

import (
	"fmt"
	"os"

	"github.com/Velocidex/go-journalctl/parser"
	kingpin "github.com/alecthomas/kingpin/v2"
	ntfs_parser "www.velocidex.com/golang/go-ntfs/parser"
)

var (
	info_command = app.Command(
		"info", "Dump information.")

	info_command_file_arg = info_command.Arg(
		"file", "The journal file to inspect",
	).Required().OpenFile(os.O_RDONLY, os.FileMode(0666))
)

func doInfo() {
	reader, _ := ntfs_parser.NewPagedReader(
		getReader(*info_command_file_arg), 1024, 10000)

	journal, err := parser.OpenFile(reader)
	kingpin.FatalIfError(err, "Can not open filesystem")

	fmt.Printf("%v\n", journal.DebugString())
}

func init() {
	command_handlers = append(command_handlers, func(command string) bool {
		switch command {
		case "info":
			doInfo()
		default:
			return false
		}
		return true
	})
}
