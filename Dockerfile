# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/wzj_sign ./

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=build /out/wzj_sign /app/wzj_sign
COPY static /app/static
VOLUME ["/app/data"]
ENV PORT=8080
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/wzj_sign"]
