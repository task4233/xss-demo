FROM golang:1.17.1 AS builder

RUN groupadd -r test && useradd --no-log-init -r -g test test
USER test

WORKDIR /home/test

COPY go.mod /home/test/go.mod
RUN go mod download

COPY . /home/test
# RUN go build -o /src/main /src/
CMD ["go", "run", "/home/test/cmd/xss-demo/main.go"]

# FROM ubuntu:18.04

# WORKDIR /usr/local/bin

# RUN apt-get update && \
#     apt-get install -y firefox mysql-client

# COPY --from=builder /src/ /usr/local/bin/
# CMD ["/usr/local/bin/main"]