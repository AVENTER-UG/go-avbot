# go-avbot - the aventer bot

AVBOT is a bot for the Matrix Chat System.


## How to use it?

First we have to create a config.yaml inside of data directory that we have to mount into the container. A sample of these config can be found in our Github repository.

```bash
docker run -v ./data:/go-avbot/data:rw avhost/go-avbot:latest
```

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

### Pentest

- Penetrate a server target
- Create a report about the penetrations test result and upload it into the chat room

There are still a lot of work. Currently our main focus is the AWS support.

### Wekan

- Receive Webhooks from your wekan boards

### Gitea

- Receive Webhooks from your gitea repo

### Unifi Protect

- Receive events from Unifi Protect devices
- Support Unifi Protect Alarm Manager

### Ollama AI

- Chat with ollama, it even support picture upload

![clipboard_20250417122305.bmp](vx_images/clipboard_20250417122305.bmp)
![clipboard_20250417122430.bmp](vx_images/clipboard_20250417122430.bmp)

## API Documentation

- [Matrix API](https://www.matrix.org/docs/spec/r0.0.0/client_server.html)
- [AWS API](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html)
- [OpenVAS](https://docs.greenbone.net/API/GMP/gmp-20.08.html)
