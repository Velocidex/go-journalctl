all:
	go build -o go-journalctl bin/*.go


windows:
	GOOS=windows GOARCH=amd64 \
            go build -ldflags="-s -w" \
	    -o go-journalctl.exe ./bin/*.go

generate:
	cd parser/ && binparsegen conversion.spec.yaml > journalctl_gen.go


test:
	go test -v ./...
