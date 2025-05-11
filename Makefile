PROTO_DIR = api
PROTO_OUT = internal/grpc/subpub/v1

.PHONY: gen-proto
gen-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(PROTO_OUT) \
		--go-grpc_out=$(PROTO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/subpub.proto

.PHONY: lint

lint:
	golangci-lint run

stop:
	docker compose stop

unit:
	go test -v ./...

run: build
	docker compose up -d --build

build:
	docker build -t subpub .