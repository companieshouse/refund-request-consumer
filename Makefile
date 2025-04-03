.EXPORT_ALL_VARIABLES:
# Common
BIN          = refund-request-consumer
VERSION		 = unversioned
# Go
CGO_ENABLED  = 1
XUNIT_OUTPUT = test.xml
LINT_OUTPUT  = lint.txt
TESTS      	 = ./...
COVERAGE_OUT = coverage.out
GO111MODULE  = on

.PHONY:
arch:
	@echo OS: $(GOOS) ARCH: $(GOARCH)
.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build: arch fmt
ifeq ($(shell uname; uname -p), Darwin arm)
	GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build --ldflags '-linkmode external -extldflags "-static"'
else
	go build
endif

.PHONY: arm-build
arm-build:
ifeq ($(shell uname; uname -p), Darwin arm)
	@make clean
	unset CC; unset CXX; CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build
endif

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit: arm-build
	@go test $(TESTS) -run 'Unit'

.PHONY: test-integration
test-integration: arm-build
	@go test $(TESTS) -run 'Integration'

.PHONY: test-with-coverage
test-with-coverage: arm-build
	go get github.com/hexira/go-ignore-cov
	go build -o ${GOBIN} github.com/hexira/go-ignore-cov
	go test -coverpkg=./... -coverprofile=$(COVERAGE_OUT) $(TESTS)
	go-ignore-cov --file $(COVERAGE_OUT)
	go tool cover -func $(COVERAGE_OUT)
	make coverage-html

.PHONY: clean-coverage
clean-coverage:
	@rm -f $(COVERAGE_OUT) coverage.html

.PHONY: coverage-html
coverage-html:
	@go tool cover -html=$(COVERAGE_OUT) -o coverage.html

.PHONY: clean
clean: clean-coverage
	go mod tidy
	rm -f ./$(BIN) ./$(BIN)-*.zip

.PHONY: package
package:
ifndef VERSION
	$(error No version given. Aborting)
endif
	$(eval tmpdir := $(shell mktemp -d build-XXXXXXXXXX))
	cp ./$(BIN) $(tmpdir)/$(BIN)
	cp ./docker_start.sh $(tmpdir)/docker_start.sh
	cd $(tmpdir) && zip ../$(BIN)-$(VERSION).zip $(BIN) docker_start.sh
	rm -rf $(tmpdir)

.PHONY: dist
dist: clean build package

.PHONY: lint
lint:
	GO111MODULE=off
	go get -u github.com/lint/golint
	golint ./... > $(LINT_OUTPUT)

.PHONY: security-check
security-check dependency-check:
	@go get golang.org/x/vuln/cmd/govulncheck
	@go build -o ${GOBIN} golang.org/x/vuln/cmd/govulncheck
	@govulncheck ./...

.PHONY: docker-image
docker-image: dist
ifneq ($(strip $(shell grep -e image: $"$$(find $"$$(mdfind "kMDItemKind == 'Folder' && kMDItemFSName == 'docker-chs-development'")$" -name $(BIN).docker-compose.yaml)$" | wc -l)),1)
	@echo "Found >1 local docker-chs-development repository, please build your image manually"
else ifeq ($(strip $(shell echo $"$$(find $"$$(mdfind "kMDItemKind == 'Folder' && kMDItemFSName == 'docker-chs-development'")$" -name $(BIN).docker-compose.yaml)$")),)
	@echo "Couldn't find compose file for this service"
else ifeq ($(shell uname; uname -p), Darwin arm)
	docker build -t $(shell image_name=$"$$(grep -e image: $"$$(find $"$$(mdfind "kMDItemKind == 'Folder' && kMDItemFSName == 'docker-chs-development'")$" -name $(BIN).docker-compose.yaml)$" | cut -d ":" -f 2 | cut -w -f 2)$"; echo $${image_name};) .
else
	@echo Unsupported OS
endif