.PHONY: test coverage publish

test:
	go test -v -race -count=1 ./...

coverage:
	gopherbadger -md="README.md"

publish:
	@latest=$$(git describe --tags --abbrev=0) && \
	GOPROXY=https://proxy.golang.org go list -m github.com/fuskovic/env@$$latest
