package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/google/uuid"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type Notification struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	ResourceID string `json:"resourceId"`
}

type FileInfo struct {
	FileName   string `json:"fileName"`
	ResourceID string `json:"resourceId"`
}

var (
	pubsubClient *pubsub.Client
	topic        *pubsub.Topic
	webhookURL   string
)

func main() {
	ctx := context.Background()

	log.Println("Starting the Google Drive Watcher Service...")

	// Get the credentials file path from the environment variable
	credsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsFile == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS environment variable must be set")
	}
	log.Printf("Using credentials from: %s", credsFile)

	// Initialize Google Drive service with service account credentials
	driveService, err := drive.NewService(ctx, option.WithCredentialsFile(credsFile))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	// Log the service account being used
	log.Printf("Using service account: %s", credsFile)

	// Get configuration from environment variables
	folderID := os.Getenv("DRIVE_FOLDER_ID")
	topicID := os.Getenv("PUBSUB_TOPIC")
	webhookURL = os.Getenv("WEBHOOK_URL")
	projectID := os.Getenv("PROJECT_ID")

	// Check if all required environment variables are set
	if folderID == "" || topicID == "" || webhookURL == "" || projectID == "" {
		log.Fatal("Environment variables DRIVE_FOLDER_ID, PUBSUB_TOPIC, WEBHOOK_URL, and PROJECT_ID must be set")
	}

	log.Printf("Environment variables:\nDRIVE_FOLDER_ID=%s\nPUBSUB_TOPIC=%s\nWEBHOOK_URL=%s\nPROJECT_ID=%s\n", folderID, topicID, webhookURL, projectID)

	// Generate a unique channel ID
	channelID := uuid.New().String()

	// Create the watch request
	watchRequest := &drive.Channel{
		Id:      channelID,
		Type:    "web_hook",
		Address: webhookURL,
		Token:   topicID,
	}

	// Set up the watch on the folder
	_, err = driveService.Files.Watch(folderID, watchRequest).Do()
	if err != nil {
		log.Fatalf("Unable to set up watch: %v. Please check if the folder ID is correct and the service account has access to the folder.", err)
	}
	log.Println("Watch set up successfully")

	// Initialize Pub/Sub client
	pubsubClient, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create Pub/Sub client: %v", err)
	}
	topic = pubsubClient.Topic(topicID)

	go func() {
		for {
			listFiles(ctx, driveService, folderID)
			time.Sleep(5 * time.Second) // Wait for 5 seconds before listing files again
		}
	}()

	// Start HTTP server for handling notifications and health checks
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request at /")

		// Log headers and body for debugging
		for name, values := range r.Header {
			for _, value := range values {
				log.Printf("Header: %s: %s", name, value)
			}
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}

		log.Printf("Request body: %s", string(body))

		var notification Notification
		if err := json.Unmarshal(body, &notification); err != nil {
			log.Printf("Error parsing request body: %v", err)
			http.Error(w, "Unable to parse request body", http.StatusBadRequest)
			return
		}

		log.Printf("Received notification for resource ID: %s", notification.ResourceID)

		// Retrieve the file metadata
		file, err := driveService.Files.Get(notification.ResourceID).Fields("id, name, mimeType, modifiedTime").Do()
		if err != nil {
			log.Printf("Error retrieving file metadata: %v", err)
			http.Error(w, "Unable to retrieve file metadata", http.StatusInternalServerError)
			return
		}

		log.Printf("File metadata retrieved: ID=%s, Name=%s, MimeType=%s", file.Id, file.Name, file.MimeType)

		// Check if the file is a Word document
		if file.MimeType != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" && file.MimeType != "application/msword" {
			log.Printf("File %s is not a Word document, ignoring.", file.Name)
			return
		}

		fileInfo := FileInfo{
			FileName:   file.Name,
			ResourceID: notification.ResourceID,
		}

		// Publish the file information to the Pub/Sub topic
		fileInfoBytes, err := json.Marshal(fileInfo)
		if err != nil {
			log.Printf("Error marshaling file info: %v", err)
			http.Error(w, "Unable to marshal file info", http.StatusInternalServerError)
			return
		}

		result := topic.Publish(ctx, &pubsub.Message{
			Data: fileInfoBytes,
		})

		// Block until the result is returned and log server-generated message IDs.
		id, err := result.Get(ctx)
		if err != nil {
			log.Printf("Failed to publish message: %v", err)
		}
		log.Printf("Published message with ID: %s for file: %s", id, file.Name)

		fmt.Fprintln(w, "Notification received and processed.")
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

func listFiles(ctx context.Context, driveService *drive.Service, folderID string) {
	// Get the list of files from Google Drive
	files, err := driveService.Files.List().Q(fmt.Sprintf("'%s' in parents", folderID)).Fields("files(id, name, createdTime, modifiedTime, mimeType)").Do()
	if err != nil {
		log.Printf("Error listing files: %v", err)
		return
	}

	// Log the list of files in the Drive folder
	for _, file := range files.Files {
		log.Printf("Scanning the drive.")

		// Check if the file is new (created or modified within the last five seconds)
		createdTime, err := time.Parse(time.RFC3339, file.CreatedTime)
		if err != nil {
			log.Printf("Error parsing created time: %v", err)
			continue
		}

		modifiedTime, err := time.Parse(time.RFC3339, file.ModifiedTime)
		if err != nil {
			log.Printf("Error parsing modified time: %v", err)
			continue
		}

		if time.Since(createdTime) <= 5*time.Second || time.Since(modifiedTime) <= 5*time.Second {
			// Check if the file is a Word document
			// if file.MimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" || file.MimeType == "application/msword" {
			log.Printf("File ID: %s, Name: %s, Created Time: %s, Modified Time: %s is a new or modified file!!!", file.Id, file.Name, file.CreatedTime, file.ModifiedTime)
			fileInfo := FileInfo{
				FileName:   file.Name,
				ResourceID: file.Id,
			}

			// Publish the file information to the Pub/Sub topic
			fileInfoBytes, err := json.Marshal(fileInfo)
			if err != nil {
				log.Printf("Error marshaling file info: %v", err)
				continue
			}

			result := topic.Publish(ctx, &pubsub.Message{
				Data: fileInfoBytes,
			})

			// Block until the result is returned and log server-generated message IDs.
			id, err := result.Get(ctx)
			if err != nil {
				log.Printf("Failed to publish message: %v", err)
			}
			log.Printf("Published message with ID: %s for file: %s", id, file.Name)

			// } else {
			// 	log.Printf("File ID: %s, Name: %s is not a Word document, ignoring.", file.Id, file.Name)
			// }
		}
	}
}
