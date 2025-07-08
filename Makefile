default: build

.PHONY: build
build:
	mkdir -p ./bin/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/ec_check.linux.amd64 .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ./bin/ec_check.linux.arm64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ./bin/ec_check.macos.amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ./bin/ec_check.macos.arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/ec_check.windows.amd64.exe .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -o ./bin/ec_check.windows.arm64.exe .
