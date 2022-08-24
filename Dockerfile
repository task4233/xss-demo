FROM golang:1.17.1 AS builder

WORKDIR /src
COPY go.mod /src/go.mod
RUN go mod download

COPY . /src/
# RUN go build -o /src/main /src/
CMD ["go", "run", "cmd/xss-demo/main.go"]

# FROM ubuntu:18.04

# WORKDIR /usr/local/bin

# RUN apt-get update && \
#     apt-get install -y firefox mysql-client

# COPY --from=builder /src/ /usr/local/bin/
# CMD ["/usr/local/bin/main"]