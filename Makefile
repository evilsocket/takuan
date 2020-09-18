all: takuan
	@ls -la build

takuan:
	@mkdir -p build
	@go build -tags musl -o build/takuan cmd/takuan/*.go

clean:
	@rm -rf build