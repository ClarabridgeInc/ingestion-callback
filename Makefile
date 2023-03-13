PROTO_RELEASE_VERSION=0.7.0
PB_DIR_NAME=protobuf-definitions-${PROTO_RELEASE_VERSION}
GO_MODULE_NAME=github.com/ClarabridgeInc/ingestion-callback

get-proto-release:
	git clone -b v${PROTO_RELEASE_VERSION} --single-branch https://github.com/ClarabridgeInc/protobuf-definitions ${PB_DIR_NAME}

proto: get-proto-release
	@go get google.golang.org/protobuf
	@$(MAKE) -C ${PB_DIR_NAME} gen-go PROTO_RELEASE_VERSION=$(PROTO_RELEASE_VERSION) GO_MODULE_NAME=$(GO_MODULE_NAME)