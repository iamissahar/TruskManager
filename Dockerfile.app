FROM golang:1.22.5-bullseye AS cert-installer

WORKDIR /app

FROM golang:1.22.5-bullseye AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" go build -o bin .

FROM builder AS final

CMD ["/app/bin"]