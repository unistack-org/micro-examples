package main

//go:generate sh -c "protoc -I./proto -I. -I$(go list -f '{{ .Dir }}' -m go.unistack.org/micro-proto/v3) --go_out=paths=source_relative:./proto proto/example.proto"
