.PHONY: build docker fake_ec test lint

build:
	go build -o bin/memcachedbetween .

docker:
	docker-compose up

fake_ec:
	fake_elasticache --servers "localhost|127.0.0.1|11213,localhost|127.0.0.1|11214,localhost|127.0.0.1|11215

test:
	go test -count 1 -race ./...

lint:
	GOGC=75 golangci-lint run --timeout 10m --concurrency 32 -v -E golint ./...
