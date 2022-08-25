FROM golang:1.17.1 AS builder

ENV USER coke
RUN groupadd -r $USER && useradd --no-log-init -r -g $USER $USER
USER $USER

WORKDIR /home/$USER

COPY go.mod /home/$USER/go.mod
RUN go mod download

COPY . /home/$USER
# RUN go build -o /src/main /src/
CMD ["go", "run", "/home/${USER}/cmd/xss-demo/main.go"]

# FROM ubuntu:18.04

# WORKDIR /usr/local/bin

# RUN apt-get update && \
#     apt-get install -y firefox mysql-client

# COPY --from=builder /src/ /usr/local/bin/
# CMD ["/usr/local/bin/main"]
