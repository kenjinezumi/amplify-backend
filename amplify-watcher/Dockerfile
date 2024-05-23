# Use the official Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
FROM golang:1.20 as builder

# Create and change to the app directory.
WORKDIR /app

# Copy go mod and sum files.
COPY go.mod .
COPY go.sum .

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed.
RUN go mod download

# Copy the source code.
COPY . .

# Build the Go app.
RUN go build -v -o main .

# Use a smaller base image for the final build.
# Use distroless as minimal base image to package the binary.
# Refer to more examples here: https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/base-debian10

# Copy the binary from the builder.
COPY --from=builder /app/main /main

# Run the web service on container startup.
CMD ["/main"]
