package amplify

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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

func processData(data []byte) ([]byte, error) {
	// Dummy function to process the data
	// For now, it just returns the same data
	return data, nil
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

// AmplifyFunction is the Google Cloud Function handler
func AmplifyFunction(ctx context.Context, e PubSubMessage) error {
	var m PubSubMessage
	if err := json.Unmarshal([]byte(e.Data), &m); err != nil {
		log.Printf("Error parsing PubSub message: %v", err)
		return err
	}

	fileID := m.Data
	inputFolderID := os.Getenv("INPUT_FOLDER_ID")
	tempFolderID := os.Getenv("TEMP_FOLDER_ID")
	outputFolderID := os.Getenv("OUTPUT_FOLDER_ID")

	if inputFolderID == "" || tempFolderID == "" || outputFolderID == "" {
		return fmt.Errorf("Folder IDs are not set")
	}

	// Retrieve the file metadata
	file, err := driveService.Files.Get(fileID).Fields("parents, name, mimeType").Do()
	if err != nil {
		log.Printf("Failed to get file metadata: %v", err)
		return err
	}

	// Check if the file is in the input folder
	inInputFolder := false
	for _, parent := range file.Parents {
		if parent == inputFolderID {
			inInputFolder = true
			break
		}
	}

	if !inInputFolder {
		log.Printf("File %s is not in the input folder, ignoring.", fileID)
		return fmt.Errorf("file is not in the input folder")
	}

	// Move the file to the temporary folder
	if err := moveFile(fileID, tempFolderID); err != nil {
		log.Printf("Failed to move file to temp folder: %v", err)
		return err
	}

	// Download the file content from the temporary folder
	resp, err := driveService.Files.Get(fileID).Download()
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return err
	}
	defer resp.Body.Close()

	fileData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read file content: %v", err)
		return err
	}

	// Process the file data
	processedData, err := processData(fileData)
	if err != nil {
		log.Printf("Failed to process file: %v", err)
		return err
	}

	// Save the processed data to a temporary file
	tempFileName := fmt.Sprintf("/tmp/%s", file.Name)
	err = ioutil.WriteFile(tempFileName, processedData, 0644)
	if err != nil {
		log.Printf("Failed to write processed data to temporary file: %v", err)
		return err
	}

	// Upload the processed file to the output folder
	fileMetadata := &drive.File{
		Name:    file.Name,
		Parents: []string{outputFolderID},
	}

	newFile, err := driveService.Files.Create(fileMetadata).Media(fileData).Do()
	if err != nil {
		log.Printf("Failed to upload processed file: %v", err)
		return err
	}

	log.Printf("Uploaded processed file: %s", newFile.Id)

	log.Printf("File %s processed successfully", fileID)
	return nil
}
