package main

import (
	"io"
	"os"

	kingpin "github.com/alecthomas/kingpin/v2"
)

type CommandHandler func(command string) bool

var (
	app = kingpin.New("goole",
		"A tool for inspecting linux journal files.")

	verbose_flag = app.Flag(
		"verbose", "Show verbose information").Bool()

	command_handlers []CommandHandler
)

func main() {
	app.HelpFlag.Short('h')
	app.UsageTemplate(kingpin.CompactUsageTemplate)
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	for _, command_handler := range command_handlers {
		if command_handler(command) {
			break
		}
	}
}

func getReader(reader io.ReaderAt) io.ReaderAt {
	return reader
}
