.PHONY: run-server run-provider

run-server:
	go run cmd/orchestrator/main.go

run-provider:
	go run cmd/provider/main.go
