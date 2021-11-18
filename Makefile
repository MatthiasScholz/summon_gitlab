app_name = summon-gitlab

include .env
export


init:
	go mod init $(app_name)

dep:
	go mod tidy

version := $(shell git rev-list -1 HEAD)
build:
	go build -o $(app_name) -ldflags "-X main.Version=$(version)"

run: build
	./$(app_name) smoketest

test-summon: build
	summon --provider ./summon-gitlab printenv SMOKE
	summon --provider ./summon-gitlab printenv PASSPHRASE
	summon --provider ./summon-gitlab printenv PASSWORD
	summon --provider ./summon-gitlab printenv PFX

clean:
	go clean

test:
	go test -v -coverprofile=app.coverage
	go tool cover -html app.coverage

docker-build:
	docker build -t $(app_name) .

docker-run: docker-build
	docker run --env-file=.env --volume ${PWD}/secrets.yml:/mnt/secrets.yml $(app_name) | grep SMOKE

docker-exec:
	docker run -it --env-file=.env --volume ${PWD}/secrets.yml:/mnt/secrets.yml --entrypoint /bin/sh $(app_name)

vault_file := certificates/PD/testuserpos1.p12
curl-store:
	curl -L -s https://freeway.porsche.org/enablement/gitlab-vault/-/raw/main/$(vault_file)
