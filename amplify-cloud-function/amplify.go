package amplify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	driveService *drive.Service
)

func init() {
	functions.CloudEvent("AmplifyFunction", AmplifyFunction)

	ctx := context.Background()
	// Use default application credentials
	service, err := drive.NewService(ctx, option.WithScopes(drive.DriveScope))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}
	driveService = service
}

type PubSubMessage struct {
	FileName   string `json:"fileName"`
	ResourceID string `json:"resourceId"`
}

func processFile(fileID string) error {
	time.Sleep(2 * time.Second)
	log.Printf("Processing file %s", fileID)
	return nil
}

func moveFile(fileID, folderID, driveID string) error {
	file, err := driveService.Files.Get(fileID).SupportsAllDrives(driveID != "").Fields("parents").Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve file %v: %v", fileID, err)
	}
	log.Printf("File parents before moving: %v", file.Parents)
	if len(file.Parents) == 0 {
		return fmt.Errorf("file %v does not have any parents", fileID)
	}
	previousParents := file.Parents
	_, err = driveService.Files.Update(fileID, nil).SupportsAllDrives(driveID != "").RemoveParents(previousParents[0]).AddParents(folderID).Do()
	if err != nil {
		return fmt.Errorf("unable to move file %v to folder %v: %v", fileID, folderID, err)
	}
	log.Printf("File %s moved to folder %s successfully", fileID, folderID)
	return nil
}

func listFilesInInputFolder(ctx context.Context, folderID, driveID string) error {
	log.Printf("Listing files in the input folder: %s", folderID)
	query := fmt.Sprintf("'%s' in parents and trashed = false", folderID)
	fileList, err := driveService.Files.List().Q(query).SupportsAllDrives(driveID != "").Fields("files(id, name, parents)").Do()
	if err != nil {
		log.Printf("Failed to list files: %v", err)
		return fmt.Errorf("Failed to list files: %v", err)
	}
	for _, file := range fileList.Files {
		log.Printf("File ID: %s, Name: %s, Parents: %v", file.Id, file.Name, file.Parents)
	}
	if len(fileList.Files) == 0 {
		log.Printf("No files found in the input folder: %s", folderID)
	}
	return nil
}

// AmplifyFunction is triggered by Pub/Sub messages.
func AmplifyFunction(ctx context.Context, e event.Event) error {
	log.Printf("Event data: %s", string(e.Data()))

	var m struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}
	if err := json.Unmarshal(e.Data(), &m); err != nil {
		log.Printf("Failed to unmarshal event data: %v", err)
		return fmt.Errorf("Failed to unmarshal event data: %v", err)
	}

	// Decode the Base64-encoded data
	decodedData, err := base64.StdEncoding.DecodeString(m.Message.Data)
	if err != nil {
		log.Printf("Failed to decode data: %v", err)
		return fmt.Errorf("Failed to decode data: %v", err)
	}

	var msg PubSubMessage
	if err := json.Unmarshal(decodedData, &msg); err != nil {
		log.Printf("Failed to unmarshal decoded data: %v", err)
		return fmt.Errorf("Failed to unmarshal decoded data: %v", err)
	}

	log.Printf("Received Pub/Sub message: %v", msg)

	fileID := msg.ResourceID
	inputFolderID := os.Getenv("INPUT_FOLDER_ID")
	tempFolderID := os.Getenv("TEMP_FOLDER_ID")
	outputFolderID := os.Getenv("OUTPUT_FOLDER_ID")
	driveID := os.Getenv("DRIVE_ID")

	if inputFolderID == "" || tempFolderID == "" || outputFolderID == "" {
		return fmt.Errorf("Folder IDs are not set")
	}

	// Log folder IDs
	log.Printf("Input Folder ID: %s, Temp Folder ID: %s, Output Folder ID: %s, Drive ID: %s", inputFolderID, tempFolderID, outputFolderID, driveID)

	// List all files in the input folder
	if err := listFilesInInputFolder(ctx, inputFolderID, driveID); err != nil {
		log.Printf("Failed to list files in the input folder: %v", err)
		return err
	}

	log.Printf("Retrieving metadata for file: %s", fileID)
	file, err := driveService.Files.Get(fileID).SupportsAllDrives(driveID != "").Fields("id, name, parents").Do()
	if err != nil {
		log.Printf("Failed to get file metadata: %v", err)
		return fmt.Errorf("Failed to get file metadata: %v", err)
	}
	log.Printf("File metadata: ID=%s, Name=%s, Parents=%v", file.Id, file.Name, file.Parents)

	inInputFolder := false
	for _, parent := range file.Parents {
		if parent == inputFolderID {
			inInputFolder = true
			break
		}
	}

	if !inInputFolder {
		log.Printf("File %s is not in the input folder, ignoring.", fileID)
		return fmt.Errorf("File is not in the input folder")
	}

	log.Printf("Moving file %s to temp folder %s", fileID, tempFolderID)
	if err := moveFile(fileID, tempFolderID, driveID); err != nil {
		log.Printf("Failed to move file to temp folder: %v", err)
		return fmt.Errorf("Failed to move file to temp folder: %v", err)
	}

	log.Printf("Processing file %s", fileID)
	if err := processFile(fileID); err != nil {
		log.Printf("Failed to process file: %v", err)
		return fmt.Errorf("Failed to process file: %v", err)
	}

	log.Printf("Downloading file %s", fileID)
	res, err := driveService.Files.Get(fileID).SupportsAllDrives(driveID != "").Download()
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return fmt.Errorf("Failed to download file: %v", err)
	}
	defer res.Body.Close()
	fileData, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("Failed to read file data: %v", err)
		return fmt.Errorf("Failed to read file data: %v", err)
	}

	log.Printf("Creating file %s in output folder %s", file.Name, outputFolderID)
	fileMetadata := &drive.File{
		Name:    file.Name,
		Parents: []string{outputFolderID},
	}
	_, err = driveService.Files.Create(fileMetadata).Media(bytes.NewReader(fileData)).SupportsAllDrives(driveID != "").Do()
	if err != nil {
		log.Printf("Failed to create file in output folder: %v", err)
		return fmt.Errorf("Failed to create file in output folder: %v", err)
	}

	log.Printf("File %s processed successfully", fileID)
	return nil
}
