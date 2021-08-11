.minimal.makefile:
	curl -fsSL -o $@ https://gitlab.com/bsm/misc/raw/master/make/go/minimal.makefile

include .minimal.makefile

proto: internal/testpb/testpb.pb.go

%.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $<
