# ===== build stage ====
FROM golang:1.21-bullseye as builder

WORKDIR /app

COPY ./go.sum ./go.sum
COPY ./go.mod ./go.mod

RUN --mount=type=cache,target=/go-mod-cache \
    go mod download

COPY . .

RUN --mount=type=cache,target=/gomod-cache \
    --mount=type=cache,target=/go-cache \
    go build -trimpath -ldflags="-w -s" -tags timetzdata -o ./bin/main ./functions/main.go

# ===== deploy stage ====
FROM golang:1.21-bullseye as deploy

RUN apt update -y
RUN apt install -y chromium

WORKDIR /app

COPY --from=builder /app/bin/main ./main

ENV BROWSER_PATH=/usr/bin/chromium

CMD ["/app/main"]
