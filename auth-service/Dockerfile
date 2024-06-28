# Use the official Golang image as the build stage
FROM golang:1.22.4-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN go build -o main .

# Use a minimal image for the final stage
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/main .
COPY .env .env
EXPOSE 8080
CMD ["./main"]
