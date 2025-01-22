IMAGE ?= ghcr.io/alexandreroman/blink1-server

all: build

build:
	docker build -t $(IMAGE) .

sync:
	vendir sync
	chmod +x vendor/linux-amd64/blink1-tool
	chmod +x vendor/linux-arm64/blink1-tool

clean:
	rm -rf vendor
