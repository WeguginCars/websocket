include .env
export $(shell sed 's/=.*//' .env)

CURRENT_DIR=$(shell pwd)
PDB_URL := postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_PORT)/$(DB_NAME)?sslmode=disable

proto-gen:
	./scripts/gen-proto.sh ${CURRENT_DIR}

mig-up:
	migrate -path migrations -database '${PDB_URL}' -verbose up

mig-down:
	migrate -path migrations -database '${PDB_URL}' -verbose down

mig-force:
	migrate -path migrations -database '${PDB_URL}' -verbose force 1

create_mig:
	@echo "Enter file name: "; \
	read filename; \
	migrate create -ext sql -dir migrations -seq $$filename

swag:
	~/go/bin/swag init -g ./api/router.go -o ./api/docs


	
run:
	go run cmd/main.go
git:
	@echo "Enter commit name: "; \
	read commitname; \
	git add .; \
	git commit -m "$$commitname"; \
	if ! git push origin main; then \
		echo "Push failed. Attempting to merge and retry..."; \
		$(MAKE) git-merge; \
		git add .; \
		git commit -m "$$commitname"; \
		git push origin main; \
	fi

git-merge:
	git fetch origin; \
	git merge origin/main

google:
	mkdir -p protos/google/api
	mkdir -p protos/protoc-gen-openapiv2/options
	curl -o protos/google/api/annotations.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto
	curl -o protos/google/api/http.proto https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto
	curl -o protos/protoc-gen-openapiv2/options/annotations.proto https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/main/protoc-gen-openapiv2/options/annotations.proto
	curl -o protos/protoc-gen-openapiv2/options/openapiv2.proto https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/main/protoc-gen-openapiv2/options/openapiv2.proto


PROTO_DIR=protos
GEN_DIR=genproto

proto:
	rm -rf $(GEN_DIR)/*
	mkdir -p $(GEN_DIR)
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/**/*.proto