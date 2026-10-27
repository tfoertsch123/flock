test:
	go test -coverprofile=cover.out

cover: test
	go tool cover -html=cover.out

doc:
	pgksite -http=127.0.0.1:6060

# update, commit and push VERSION before running make tag
tag:	VERSION
	git fetch origin && \
	[ "$$(git branch --show-current)" = master ] && \
	L="$$(git rev-parse HEAD)" && \
	R="$$(git rev-parse refs/remotes/origin/master)" && \
	[ "$$L" = "$$R" ] && \
	T="$$(cat VERSION)" && \
	git tag $$T && \
	git push origin tag $$T && \
	M="$$(sed '/^module/!d; s/^module *//' go.mod)" && \
	GOPROXY=proxy.golang.org go list -m $$M@$$T
