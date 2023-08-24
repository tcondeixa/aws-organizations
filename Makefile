.PHONY: dev lint test test.coverage coverage benchmark docs clean

default: test

dev:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint:
	golangci-lint run -v --enable gofmt

test:
	go test -v -race ./...

test.coverage:
	go test -v -race -cover -coverprofile=coverage.out ./...

coverage: test.coverage
	go tool cover -func=coverage.out -o=coverage_summary.out

benchmark:
	go test -bench=. -count=2 -run=^$

docs:
	go doc -u -all

clean:
	rm -rf coverage.out coverage_summary.out
