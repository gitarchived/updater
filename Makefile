run:
	@go run cmd/updater/main.go $(filter-out $@,$(MAKECMDGOALS))
build:
	@go build -o bin/updater cmd/updater/main.go
genenv:
	@sh scripts/genenv.sh
