#Dockerfile vars

#vars
IMAGENAME=go-avbot
TAG=v0.6.3
BRANCH=${TAG}
BRANCHSHORT=$(shell echo ${BRANCH} | awk -F. '{ print $$1"."$$2 }')
IMAGEFULLNAME=avhost/${IMAGENAME}
LASTCOMMIT=$(shell git log -1 --pretty=short | tail -n 1 | tr -d " " | tr -d "UPDATE:")
BUILDDATE=${shell date -u +%Y%m%dT%H%M%SZ}

.PHONY: help build bootstrap all docs publish push version

build:
	@echo ">>>> Build docker image"
	@docker build --build-arg TAG=${TAG} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .

build-bin:
	@echo ">>>> Build binary"
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.BuildVersion=${BUILDDATE} -X main.GitVersion=${TAG} -extldflags \"-static\"" .

push:
	@echo ">>>> Publish docker image: " ${BRANCH} ${BRANCHSHORT}
	-docker buildx create --use --name buildkit
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCH} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:${BRANCHSHORT} .
	@docker buildx build --sbom=true --provenance=true --platform linux/amd64 --push --build-arg TAG=${BRANCH} --build-arg BUILDDATE=${BUILDDATE} -t ${IMAGEFULLNAME}:latest .
	-docker buildx rm buildkit

update-gomod:
	go get -u
	go mod tidy

sboom:
	syft dir:. > sbom.txt
	syft dir:. -o json > sbom.json

seccheck:
	grype --add-cpes-if-none .

imagecheck:
	grype --add-cpes-if-none ${IMAGEFULLNAME}:latest > cve-report.md

all: build seccheck imagecheck sboom
