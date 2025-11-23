# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN go build -o server ./cmd/server

# Final stage
FROM alpine:latest

# Install tzdata for timezone support
RUN apk add --no-cache tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .
# Copy the plan file
COPY --from=builder /app/plan.md .

# Expose port 8080
EXPOSE 8080

# Run the server
CMD ["./server"]
