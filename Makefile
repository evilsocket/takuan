all: takuan
	@ls -la build

takuan:
	@mkdir -p build
	@go build -tags musl -o build/takuan cmd/takuan/*.go

composer_build:
	@docker-compose build

composer_up: 
	@docker-compose up

clean:
	@rm -rf build