FROM golang:alpine as builder

RUN apk update && apk add --no-cache git make gcc libc-dev

# download, cache and install deps
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# copy and compiled the app
COPY . .
RUN make takuan

# install private key and download public key for github.com
ARG SSH_PRIVATE_KEY
RUN mkdir -p -m 0600 /root/.ssh/
RUN echo "${SSH_PRIVATE_KEY}" > /root/.ssh/id_rsa
RUN ssh-keyscan github.com >> ~/.ssh/known_hosts

# start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# copy the prebuilt binary from the builder stage
COPY --from=builder /app/build/takuan .

CMD ["./takuan", "-config", "/etc/takuan/config.yml"]