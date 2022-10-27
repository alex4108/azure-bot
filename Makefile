PHONY: .build .docker .docker-release .test

ifndef AZURE_BOT_TAG
override AZURE_BOT_TAG = azure-bot
endif

.DEFAULT_GOAL := build

build:
	go mod tidy
	go build -o ./bin/azure-bot

docker: build
	docker build -t $(AZURE_BOT_TAG) .

docker-release: build
	docker buildx build --platform linux/amd64 -t alex4108/azure-bot:$(AZURE_BOT_TAG) --push .

test: docker
	docker run --rm -e CI=true $(AZURE_BOT_TAG)
