
build:
	go build -o director cmd/director/main.go

lint:
	go get -u golang.org/x/lint/golint
	$$(go list -f {{.Target}} golang.org/x/lint/golint) -set_exit_status ./...

cleanup:
	go mod tidy
	go fmt ./...
	$(MAKE) lint

.PHONY: cleanup
