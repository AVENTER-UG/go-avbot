FROM golang:alpine AS builder

WORKDIR /build

ARG BUILDDATE ""

COPY . /build/

RUN apk update && apk upgrade && apk add git gcc musl-dev && \
    go get -d

RUN CGO_CFLAGS="-D_LARGEFILE64_SOURCE -g -O2 -Wno-return-local-addr" CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.MinVersion=${BUILDDATE} -extldflags \"-static\"" -o main .

FROM alpine
LABEL maintainer="Andreas Peters <support@aventer.biz>"
LABEL org.opencontainers.image.title="go-avbot"
LABEL org.opencontainers.image.description="Matrix Bot"
LABEL org.opencontainers.image.vendor="AVENTER UG (haftungsbeschränkt)"
LABEL org.opencontainers.image.source="https://github.com/AVENTER-UG/"

ENV BIND_ADDRESS=:4050 DATABASE_TYPE=sqlite3 DATABASE_URL=/app/data/go-neb.db?_busy_timeout=5000 

RUN apk add --no-cache ca-certificates
USER nobody

COPY --from=builder /build/main /app/

EXPOSE 10000

WORKDIR "/app"

CMD ["./main"]

EXPOSE 4050

