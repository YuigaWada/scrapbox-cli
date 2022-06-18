run:
	$(MAKE) build && ./sbox
build:
	go build
format:
	go fmt
clean:
	rm -f scrapbox-cli sbox