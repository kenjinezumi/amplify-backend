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

	// Get configuration from environment variables
	folderID := os.Getenv("DRIVE_FOLDER_ID") // Unique folder identifier
	topic := os.Getenv("PUBSUB_TOPIC")       // Pubsub topic
	channelID := os.Getenv("CHANNEL_ID")     // Channel id for notifications in case there's another one
	webhookURL := os.Getenv("WEBHOOK_URL")   // Cloud function URL

	// Check if all required environment variables are set
	if folderID == "" || topic == "" || channelID == "" || webhookURL == "" {
		log.Fatal("Environment variables DRIVE_FOLDER_ID, PUBSUB_TOPIC, CHANNEL_ID, and WEBHOOK_URL must be set")
	}

	// Create the watch request
	watchRequest := &drive.Channel{
		Id:      channelID, // Unique identifier for the watch
		Type:    "web_hook",
		Address: webhookURL, // Cloud Function URL
		Token:   topic,      // Pub/Sub topic
	}

	// Set up the watch on the folder
	_, err = driveService.Files.Watch(folderID, watchRequest).Do()
	if err != nil {
		log.Fatalf("Unable to set up watch: %v", err)
	}

	fmt.Println("Watch set up successfully")
}
