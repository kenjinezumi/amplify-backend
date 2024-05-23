package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	driveService *drive.Service
)

// init initializes the Google Drive service using Application Default Credentials.
func init() {
	ctx := context.Background()

	// Use Application Default Credentials
	service, err := drive.NewService(ctx, option.WithScopes(drive.DriveScope))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	driveService = service
}

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Data string `json:"data"`
}

// processFile simulates file processing by sleeping for 2 seconds.
func processFile(fileID string) error {
	// Simulate processing time
	time.Sleep(2 * time.Second)
	// Log the processing step
	log.Printf("Processing file %s", fileID)
	return nil
}

// moveFile moves a file to a specified folder in Google Drive.
func moveFile(fileID, folderID string) error {
	// Retrieve the file metadata
	file, err := driveService.Files.Get(fileID).Fields("parents").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve file %v: %v", fileID, err)
	}

	// Remove the file from its current parents
	previousParents := file.Parents
	_, err = driveService.Files.Update(fileID, nil).RemoveParents(previousParents[0]).AddParents(folderID).Do()
	if err != nil {
		return fmt.Errorf("unable to move file %v to folder %v: %v", fileID, folderID, err)
	}

	return nil
}

// handleRequest handles the incoming HTTP request triggered by a Pub/Sub message.
func handleRequest(w http.ResponseWriter, r *http.Request) {
	var m PubSubMessage

	// Decode the JSON payload from the request body
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fileID := m.Data
	tempFolderID := os.Getenv("TEMP_FOLDER_ID")     // Set these as environment variables
	outputFolderID := os.Getenv("OUTPUT_FOLDER_ID") // Set these as environment variables

	if tempFolderID == "" || outputFolderID == "" {
		http.Error(w, "Folder IDs are not set", http.StatusInternalServerError)
		return
	}

	// Move to temporary folder
	if err := moveFile(fileID, tempFolderID); err != nil {
		log.Printf("Failed to move file to temp folder: %v", err)
		http.Error(w, fmt.Sprintf("Failed to move file to temp folder: %v", err), http.StatusInternalServerError)
		return
	}

	// Process the file
	if err := processFile(fileID); err != nil {
		log.Printf("Failed to process file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process file: %v", err), http.StatusInternalServerError)
		return
	}

	// Move to output folder
	if err := moveFile(fileID, outputFolderID); err != nil {
		log.Printf("Failed to move file to output folder: %v", err)
		http.Error(w, fmt.Sprintf("Failed to move file to output folder: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("File %s processed successfully", fileID)
	fmt.Fprintln(w, "File processed successfully")
}

// main is the entry point for the Cloud Function.
func main() {
	http.HandleFunc("/", handleRequest)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
