package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// Initialize Google Drive service with default credentials
	driveService, err := drive.NewService(ctx, option.WithScopes(drive.DriveScope))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// Get the folder ID and Pub/Sub topic from environment variables
	folderID := os.Getenv("DRIVE_FOLDER_ID")
	topic := os.Getenv("PUBSUB_TOPIC")

	if folderID == "" || topic == "" {
		log.Fatalf("Environment variables DRIVE_FOLDER_ID and PUBSUB_TOPIC must be set")
	}

	// Create the watch request
	watchRequest := &drive.Channel{
		Id:      "unique-channel-id", // Unique identifier for the watch
		Type:    "web_hook",
		Address: "https://your_cloud_function_url", // Replace with your Cloud Function URL
		Token:   topic,                             // Pub/Sub topic
	}

	// Set up the watch on the folder
	_, err = driveService.Files.Watch(folderID, watchRequest).Do()
	if err != nil {
		log.Fatalf("Unable to set up watch: %v", err)
	}

	fmt.Println("Watch set up successfully")
}
