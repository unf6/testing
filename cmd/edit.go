package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"path/filepath"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/unf6/testing/pkg/database"
)

var editCmd = &cobra.Command{
	Use: "edit",
	Short: "Edit a task",
	Long: "Edit a task by providing a title and id",
    Args: cobra.MaximumNArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
	
		id, _ := cmd.Flags().GetInt("id")
		title, _ := cmd.Flags().GetString("title")
		status, _ := cmd.Flags().GetString("status")

	
		saveOptionPrompt := promptui.Select{
			Label: "Where is to save the task",
			Items: []string{"Database (sqlite)", "CSV File"},
			CursorPos: 0,
			
		}

		_, saveOption, saveOptionErr := saveOptionPrompt.Run()
		if saveOptionErr != nil {
			fmt.Printf("%s Error: %v", promptui.IconBad, saveOptionErr)
			os.Exit(1)
		}

		switch saveOption {
		case "CSV File":
			editTaskInCSV(id, title, status)
		default:
			editTaskInDatabase(id, title, status)
		}
        fmt.Printf("%s Task edited succesfully!", promptui.IconGood)
	},

}

func init() {
	rootCmd.AddCommand(editCmd)

	editCmd.Flags().Int("id", 0, "Id of the task")
	editCmd.Flags().String("title", "", "New title for the task")
	editCmd.Flags().String("status", "", "New status for the task")
}

func editTaskInDatabase(id int, title string, status string) {
	db := database.GetDB()

	if id == 0 {
		prompt := promptui.Prompt {
			Label: "Task ID",
			Validate: validateIDInput,
		}
		idInput, err := prompt.Run()

		if err != nil {

			fmt.Printf("%s Error: %v\n", promptui.IconBad, err)
			os.Exit(1)
		}
		id, _ = strconv.Atoi(idInput)
	}

	if title == "" {
		prompt := promptui.Prompt{
			Label: "New Task Title (leave blank to keep unchanged)",
		}
		title, _ = prompt.Run()
	}

	// Prompt for status if not provided
	if status == "" {
		prompt := promptui.Select{
			Label: "New Task Status",
			Items: []string{"pending", "in-progress", "completed"},
		}
		_, status, _ = prompt.Run()
	}

	// Fetch the existing task
	var currentTitle, currentStatus string
	err := db.QueryRow("SELECT title, status FROM tasks WHERE id = ?", id).Scan(&currentTitle, &currentStatus)
	if err != nil {
		fmt.Printf("Task with ID %d not found: %v\n", id, err)
		os.Exit(1)
	}

	// If no new values provided, keep the existing ones
	if title == "" {
		title = currentTitle
	}
	if status == "" {
		status = currentStatus
	}

	// Update the task in the database
	updateQuery := `
		UPDATE tasks
		SET title = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err = db.Exec(updateQuery, title, status, time.Now().UTC(), id)
	if err != nil {
		fmt.Printf("Failed to update the task: %v\n", err)
		os.Exit(1)
	}
}

func editTaskInCSV(id int, title string, status string) {

	configDir := filepath.Join(os.Getenv("HOME"), ".config", "tasks-cli")

	// Create the configuration directory if it doesn't exist
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		err = os.MkdirAll(configDir, 0755)
		if err != nil {
			fmt.Printf("Failed to create config directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Define the CSV file path
	csvFilePath := filepath.Join(configDir, "tasks.csv")

	// Open CSV file
	file, err := os.OpenFile(csvFilePath, os.O_RDWR, 0644)
	if err != nil {
		fmt.Printf("Failed to open CSV file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Read all records
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Failed to read CSV file: %v\n", err)
		os.Exit(1)
	}

	// Find and update the task
	found := false
	for i, record := range records {
		if i == 0 {
			// Skip header row
			continue
		}
		recordID, _ := strconv.Atoi(record[0])
		if recordID == id {
			found = true
			if title != "" {
				record[1] = title
			}
			if status != "" {
				record[3] = status
			}
			record[5] = time.Now().UTC().String() // Update updated_at timestamp
			records[i] = record
			break
		}
	}

	if !found {
		fmt.Printf("Task with ID %d not found in CSV file.\n", id)
		os.Exit(1)
	}

	// Write updated records back to the file
	file.Truncate(0)
	file.Seek(0, 0)
	writer := csv.NewWriter(file)
	err = writer.WriteAll(records)
	if err != nil {
		fmt.Printf("Failed to update CSV file: %v\n", err)
		os.Exit(1)
	}
	writer.Flush()
}


func validateIDInput(input string) error {
	_, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Errorf("invalid id")
	}
	return nil
}