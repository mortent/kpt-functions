.PHONY: docs fix vet fmt lint test build tidy


build:
	go build -o localpackage-fn -v .

all: fix vet fmt lint test build tidy

fix:
	go fix ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

lint:
	(which golangci-lint || go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.22.2)
	$(GOBIN)/golangci-lint run ./...

test:
	go test -cover ./...

vet:
	go vet ./...

docker:
	docker build .