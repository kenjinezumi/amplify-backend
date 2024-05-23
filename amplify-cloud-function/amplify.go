package amplify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	driveService *drive.Service
)

func init() {
	functions.HTTP("AmplifyFunction", AmplifyFunction)

	ctx := context.Background()
	// Use Application Default Credentials
	service, err := drive.NewService(ctx, option.WithScopes(drive.DriveScope))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	driveService = service

}

type PubSubMessage struct {
	Data string `json:"data"`
}

func processFile(fileID string) error {
	time.Sleep(2 * time.Second)
	log.Printf("Processing file %s", fileID)
	return nil
}

func moveFile(fileID, folderID string) error {
	file, err := driveService.Files.Get(fileID).Fields("parents").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve file %v: %v", fileID, err)
	}
	previousParents := file.Parents
	_, err = driveService.Files.Update(fileID, nil).RemoveParents(previousParents[0]).AddParents(folderID).Do()
	if err != nil {
		return fmt.Errorf("unable to move file %v to folder %v: %v", fileID, folderID, err)
	}
	return nil
}

// Google Cloud Function handler
func AmplifyFunction(w http.ResponseWriter, r *http.Request) {
	var m PubSubMessage
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fileID := m.Data
	inputFolderID := os.Getenv("INPUT_FOLDER_ID")
	tempFolderID := os.Getenv("TEMP_FOLDER_ID")
	outputFolderID := os.Getenv("OUTPUT_FOLDER_ID")

	if inputFolderID == "" || tempFolderID == "" || outputFolderID == "" {
		http.Error(w, "Folder IDs are not set", http.StatusInternalServerError)
		return
	}

	file, err := driveService.Files.Get(fileID).Fields("parents").Do()
	if err != nil {
		log.Printf("Failed to get file metadata: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get file metadata: %v", err), http.StatusInternalServerError)
		return
	}

	inInputFolder := false
	for _, parent := range file.Parents {
		if parent == inputFolderID {
			inInputFolder = true
			break
		}
	}

	if !inInputFolder {
		log.Printf("File %s is not in the input folder, ignoring.", fileID)
		http.Error(w, "File is not in the input folder", http.StatusBadRequest)
		return
	}

	if err := moveFile(fileID, tempFolderID); err != nil {
		log.Printf("Failed to move file to temp folder: %v", err)
		http.Error(w, fmt.Sprintf("Failed to move file to temp folder: %v", err), http.StatusInternalServerError)
		return
	}

	if err := processFile(fileID); err != nil {
		log.Printf("Failed to process file: %v", err)
		http.Error(w, fmt.Sprintf("Failed to process file: %v", err), http.StatusInternalServerError)
		return
	}

	if err := moveFile(fileID, outputFolderID); err != nil {
		log.Printf("Failed to move file to output folder: %v", err)
		http.Error(w, fmt.Sprintf("Failed to move file to output folder: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("File %s processed successfully", fileID)
	fmt.Fprintln(w, "File processed successfully")
}
