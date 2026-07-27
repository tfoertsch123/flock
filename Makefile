test:
	go test -coverprofile=cover.out

cover: test
	go tool cover -html=cover.out

doc:
	pgksite -http=127.0.0.1:6060
