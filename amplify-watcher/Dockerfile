# Use the official Golang image to create a build artifact.
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

# List the contents of the directory for debugging purposes
RUN ls -la

# Build the Go app.
RUN go build -v -o main .

# Use Ubuntu 22.04 as the base image for the final build to ensure the correct glibc version.
FROM ubuntu:22.04

# Install necessary packages
RUN apt-get update && apt-get install -y ca-certificates

# Create the /app directory
RUN mkdir /app

# Copy the binary from the builder.
COPY --from=builder /app/main /app/main

# Copy the service account key file into the image.
COPY ./service-account-key.json /app/service-account-key.json

# List the contents of the directory for debugging purposes.
RUN ls -la /app

# Define build arguments
ARG DRIVE_FOLDER_ID
ARG PUBSUB_TOPIC
ARG WEBHOOK_URL
ARG PROJECT_ID

# Set the environment variable for Google Cloud credentials.
ENV GOOGLE_APPLICATION_CREDENTIALS=/app/service-account-key.json
ENV DRIVE_FOLDER_ID $DRIVE_FOLDER_ID
ENV PUBSUB_TOPIC $PUBSUB_TOPIC
ENV WEBHOOK_URL $WEBHOOK_URL
ENV PROJECT_ID $PROJECT_ID

# Expose the port for Cloud Run
EXPOSE 8080

# Run the web service on container startup.
CMD ["/app/main"]
