# go-avbot - the aventer bot

## Github Repo

[https://git.aventer.biz/AVENTER](https://git.aventer.biz/AVENTER)

## Security

We verify our image automatically by clair. If you want to see the current security status, please have a look in travis-ci.

## What is go-avbot

AVBOT is our digital working partner. He is helping us with our daily business. From creating and sending out invoices to install server applications. AVBOT is based on [go-neb](https://github.com/matrix-org/go-neb), a matrix BOT developed in golang.

Our Kanban Board is [here](https://wekan.aventer.biz/b/XePZjKD4mK3eFY8MS/go-avbot)

## License

go-neb is under the Apache License. To make it more complicated, our code are under GPL. These are:

- aws (services/aws)
- invoice (services/invoice)
- pentest (services/pentest)

## Features

### AWS

- Start/Stop of AWS instances
- Show list of all instances in all regions
- Create Instances
- Search AMI's

### Ispconfig

- Create Invoice and send them out
- Show invoices of a user

### Pentest

- Penetrate a server target
- Create a report about the penetrations test result and upload it into the chat room

There are still a lot of work. Currently our main focus is the AWS support.

### Github

- Receive Webhooks from your github repositories.
- Create Issues

### Travis-CI

- Receive Webhooks from your travis account

### Wekan

- Receive Webhooks from your wekan boards

### Gitea

- Receive Webhooks from your gitea repo

### NLP (Natural Language Processing) 

- Gateway to the IKY Framework

## Software Requirements

```bash
go get github.com/sirupsen/logrus
go get github.com/matrix-org/util
go get github.com/mattn/go-sqlite3
go get github.com/prometheus/client_golang/prometheus
go get github.com/matrix-org/dugong
go get git.aventer.biz/AVENTER/gomatrix
go get github.com/mattn/go-shellwords
go get gopkg.in/yaml.v2
go get golang.org/x/oauth2
go get github.com/google/go-github/github
go get gopkg.in/alecthomas/kingpin.v2
go get github.com/russross/blackfriday
go get github.com/aws/aws-sdk-go
```

## API Documentation

- [Matrix API](https://www.matrix.org/docs/spec/r0.0.0/client_server.html)
- [AWS API](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html)
- [OpenVAS](http://docs.greenbone.net/API/OMP/omp-7.0.html)

## Changelog

### v0.0.1

- aws: add show instances of all configured regions
- aws: add start and stop of instances
- aws: add search of images
- invoice: add create of invoices
- invoice: add view invoices of a user

### v0.0.2

- pentest: give out a list of all scanner configs
- pentest: start a pentest
- pentest: get a status of the pentest
- pentest: download the report file
- aws: add run instances

### v0.0.3

- add travis webhook support (fork from the original project)
- add wekan webhook support (is a fork of the travis version)

### v0.0.4

- modify repo to git.aventer.biz
- add gitea support

### v0.0.5

- add nlp support