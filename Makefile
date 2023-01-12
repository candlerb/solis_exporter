.PHONY: all
all:
	go build -o bin/ ./...

.PHONY: test
test:
	go fmt ./...
	# https://github.com/golang/go/issues/56755
	CGO_ENABLED=0 go vet ./...
	go test -vet=off ./...

.PHONY: clean
clean:
	find . -name '*~' -delete
	rm -f bin/*

.PHONY: docs
docs: clean
	mkdocs build
