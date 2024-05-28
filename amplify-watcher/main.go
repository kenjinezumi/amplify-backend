package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	log.Println("Starting the Google Drive Watcher Service...")

	// Get the credentials file path from the environment variable
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		log.Println("GOOGLE_APPLICATION_CREDENTIALS environment variable not set, using default application credentials")
	} else {
		log.Printf("Using credentials from: %s", credsFile)
	}

	// Initialize Google Drive service with default credentials
	creds, err := google.FindDefaultCredentials(ctx, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to find default credentials: %v", err)
	}

	driveService, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// Log the service account being used
	serviceAccount := creds.ProjectID
	if serviceAccount == "" {
		serviceAccount = "unknown (default credentials used)"
	}
	log.Printf("Using service account: %s", serviceAccount)

	// Get configuration from environment variables
	folderID := os.Getenv("DRIVE_FOLDER_ID")
	topic := os.Getenv("PUBSUB_TOPIC")
	channelID := os.Getenv("CHANNEL_ID")
	webhookURL := os.Getenv("WEBHOOK_URL")

	// Check if all required environment variables are set
	if folderID == "" || topic == "" || channelID == "" || webhookURL == "" {
		log.Fatal("Environment variables DRIVE_FOLDER_ID, PUBSUB_TOPIC, CHANNEL_ID, and WEBHOOK_URL must be set")
	}

	log.Printf("Environment variables:\nDRIVE_FOLDER_ID=%s\nPUBSUB_TOPIC=%s\nCHANNEL_ID=%s\nWEBHOOK_URL=%s\n", folderID, topic, channelID, webhookURL)

	// Create the watch request
	watchRequest := &drive.Channel{
		Id:      channelID,
		Type:    "web_hook",
		Address: webhookURL,
		Token:   topic,
	}

	// Set up the watch on the folder
	_, err = driveService.Files.Watch(folderID, watchRequest).Do()
	if err != nil {
		log.Fatalf("Unable to set up watch: %v. Please check if the folder ID is correct and the service account has access to the folder.", err)
	}
	log.Println("Watch set up successfully")

	// Start HTTP server for health checks and to meet Cloud Run requirements
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request at /")
		fmt.Fprintln(w, "Watcher is active and running.")
	})

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request at /healthz")
		fmt.Fprintln(w, "ok")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port in local environment if not set
	}
	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to listen and serve: %v", err)
	}
}
s