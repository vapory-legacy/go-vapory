.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/vaporyco/go-vapory/cmd/swarm
	go build -o bin/gvap     github.com/vaporyco/go-vapory/cmd/gvap
	go build -o bin/bootnode github.com/vaporyco/go-vapory/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/vaporyco/go-vapory/swarm/...
