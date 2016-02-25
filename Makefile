default: build

BIN=mysql-log-player
BIN_x64=mysql-log-player-x64
STAGE=production

all: build build_x64

deps: .vendor

.vendor: Goopfile.lock
	goop install

generate_version:
	script/version

clean:
	rm $(BIN) $(BIN_x64) || true

build: deps generate_version
	goop exec go build -o $(BIN)

build_x64: deps generate_version
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 goop exec go build -o $(BIN_x64)

test: deps
	goop exec go test ./...

deploy:
	script/deploy $(STAGE)
