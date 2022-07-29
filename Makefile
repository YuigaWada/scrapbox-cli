run:
	$(MAKE) build && ./sbox/sbox
build:
	cd ./sbox; go build
format:
	go fmt
clean:
	rm -f scrapbox-cli sbox