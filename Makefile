all: install

lint:
	golangci-lint run --exclude 'unused'

test:
	go test -v ./... --count=1

install:
	go build -o ./build/collaborate-storage ./cmd

build-docker:
	CGO_ENABLED=0 GOOS=linux go build -o ./build/collaborate-storage ./cmd
	docker build --tag collaborate-storage:1.0 ./build/
	rm ./build/collaborate-storage
