.PHONY: build test vet

build:
	go build ./...

# -race detects data races in concurrent ring/orchestrator code.
# -count=1 disables test result caching so CI always gets a fresh run.
test:
	go test -race -count=1 ./...

vet:
	go vet ./...
