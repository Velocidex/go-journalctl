# Go parser for systemd journal log files.

The foramt is documented here https://systemd.io/JOURNAL_FILE_FORMAT/

This parser is written in pure Go (rather than binding the system C
libraries) which makes it portable to other operating systems.


## Reading a journal file

To dump a journal file use the `cat` command:

```
./go-journalctl cat /run/log/journal/4e7cbddbe9494fb9876af4e3e85c9eb4/system.journal
```

To follow for new entries use the `-f` flag

```
./go-journalctl cat -f /run/log/journal/4e7cbddbe9494fb9876af4e3e85c9eb4/system.journal
```

## Parsed VS. raw logs.

Internally systemd treats all entries as being strings, but many
entries are integers or timestamps. By default `go-journalctl` will
parse the events based on known event fields into two groups:

1. The `System` group contains trusted fields added by the System and
   not settable by the logging client.
2. The `EventData` field is free form and contains arbitrary fields
   and values set by the logging client.

This scheme is similar to the Windows event log scheme and makes it
eaiser to insert the data into structuted storage and perform
structured queries on the data.

If you wish to see the original `raw` event fields, set the `--raw`
flag.
