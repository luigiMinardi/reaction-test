run:
	@echo "Running..."
	@go build -o ./bin/rt ./cmd/reaction-test/reaction-test.go
	@./bin/rt
