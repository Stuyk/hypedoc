.PHONY: build test smoke readme clean

build:
	go build -trimpath -ldflags "-s -w" -o bin/hypedoc .

test:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...
	go test ./...

# End-to-end: init, build, and strip in a temp dir (needs Chrome/Chromium or network).
smoke: build
	@bin=$$(pwd)/bin/hypedoc; dir=$$(mktemp -d) && cd $$dir && \
	printf '# Smoke\n\nBody.\n' > README.md && cp README.md orig.md && \
	$$bin init && $$bin build && test -s .hypedoc/img/01-hero.light.png && \
	$$bin strip && diff README.md orig.md && echo "smoke ok" && rm -rf $$dir

readme: build
	./bin/hypedoc build --doc README.md

clean:
	rm -rf bin dist
