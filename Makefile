.PHONY: solis_exporter
solis_exporter:
	go build -o bin ./...

.PHONY: test
test:
	go fmt ./...
	# https://github.com/golang/go/issues/56755
	CGO_ENABLED=0 go vet ./...
	go test -vet=off ./...

.PHONY: clean
clean:
	find . -name '*~' -delete
	rm -f solis_exporter

.PHONY: docs
docs: clean
	mkdocs build
