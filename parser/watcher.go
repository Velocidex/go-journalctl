package parser

import (
	"context"
	"io"
	"time"

	"github.com/Velocidex/ordereddict"
)

type FileOpener func() (reader io.ReaderAt, closer func(), err error)

func WatchFile(ctx context.Context,
	opener FileOpener) chan *ordereddict.Dict {
	output_chan := make(chan *ordereddict.Dict)

	go func() {
		defer close(output_chan)

		seq := int64(-1)
		for {
		next:
			select {
			case <-ctx.Done():
				return

			case <-time.After(time.Second * 2):
				reader, closer, err := opener()
				if err != nil {
					continue
				}

				journal, err := OpenFile(reader)
				if err != nil {
					closer()
					continue
				}

				last_seq := journal.GetLastSequence()
				if seq < 0 {
					seq = int64(last_seq)
					closer()
					continue
				}

				if int64(last_seq) > seq {
					journal.MinSeq = uint64(seq)
					journal.MaxSeq = last_seq
					seq = int64(last_seq)

					row_chan := journal.GetLogs(context.Background())
					for {
						select {
						case <-ctx.Done():
							closer()
							return

						case row, ok := <-row_chan:
							if !ok {
								closer()
								break next
							}
							output_chan <- row
						}
					}
				}

			}

		}
	}()
	return output_chan
}
