build-linux:
	GOARCH=amd64 GOOS=linux go build -o nats-jetstream-linux nats-jetstream-gui

build-windows:
	GOARCH=amd64 GOOS=windows go build -o nats-jetstream-windows.exe nats-jetstream-gui

build: build-linux build-windows