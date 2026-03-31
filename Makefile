.PHONY: test publish

test:
	go test -v -race -count=1 ./...

publish:
	@latest=$$(git describe --tags --abbrev=0) && \
	GOPROXY=https://proxy.golang.org go list -m github.com/fuskovic/env@$$latest
