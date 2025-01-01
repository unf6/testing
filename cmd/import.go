package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/unf6/testing/models" // Import the models package
	"github.com/unf6/testing/pkg/database"
	"github.com/unf6/testing/pkg/utils"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
    Use:   "import [json|csv]",
    Short: "Import tasks into SQLite from CSV or JSON file",
    Long:  `Import tasks into SQLite from a CSV or JSON file. You can select the file format interactively if no arguments are provided.`,
    Run: func(cmd *cobra.Command, args []string) {
        // Determine import format
        var format string
        if len(args) > 0 {
            format = args[0]
            if format != "json" && format != "csv" {
                fmt.Println("Invalid format specified. Valid options are 'json' or 'csv'.")
                return
            }
        } else {
            formatPrompt := promptui.Select{
                Label: "Select import format",
                Items: []string{"JSON", "CSV"},
            }
            _, format, err := formatPrompt.Run()
            if err != nil {
                fmt.Printf("Error during format selection: %v\n", err)
                return
            }
            format = format
        }

        // Prompt for file path
        filePrompt := promptui.Prompt{
            Label:   "Enter import file path",
            Default: fmt.Sprintf("%s/tasks.%s", filepath.Join(os.Getenv("HOME"), ".config", "tasks-cli"), format),
        }
        filePath, err := filePrompt.Run()
        if err != nil {
            fmt.Printf("Error during file path input: %v\n", err)
            return
        }

        // Import tasks based on format
        switch format {
        case "json":
            err = importFromJSON(filePath)
        case "csv":
            err = importFromCSV(filePath)
        default:
            fmt.Println("Invalid format selected.")
            return
        }

        if err != nil {
            fmt.Printf("Error importing tasks: %v\n", err)
        } else {
            fmt.Println("Tasks imported successfully.")
        }
    },
}

// importFromJSON imports tasks from a JSON file into SQLite
func importFromJSON(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening JSON file: %v", err)
    }
    defer file.Close()

    var tasks []models.Task
    decoder := json.NewDecoder(file)
    err = decoder.Decode(&tasks)
    if err != nil {
        return fmt.Errorf("error decoding JSON file: %v", err)
    }

    db := database.GetDB()
    for _, task := range tasks {
        // Check if task with the same ID already exists
        var exists bool
        err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM tasks WHERE id = ?)`, task.ID).Scan(&exists)
        if err != nil {
            return fmt.Errorf("error checking if task exists: %v", err)
        }

        if exists {
            fmt.Printf("Task with ID %d already exists, skipping import...\n", task.ID)
            continue // Skip this task if it already exists
        }

        _, err = db.Exec(`INSERT INTO tasks (id, title, description, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
            task.ID, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt)
        if err != nil {
            return fmt.Errorf("error inserting task into database: %v", err)
        }
    }

    return nil
}

// importFromCSV imports tasks from a CSV file into SQLite
func importFromCSV(filePath string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("error opening CSV file: %v", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return fmt.Errorf("error reading CSV file: %v", err)
    }

    db := database.GetDB()
    for i, record := range records {
        if i == 0 {
            // Skip header row
            continue
        }

        createdAt, _ := time.Parse(time.RFC3339, record[4])
        updatedAt, _ := time.Parse(time.RFC3339, record[5])

        // Check if task with the same ID already exists
        var exists bool
        err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM tasks WHERE id = ?)`, utils.MustAtoi(record[0])).Scan(&exists)
        if err != nil {
            return fmt.Errorf("error checking if task exists: %v", err)
        }

        if exists {
            fmt.Printf("Task with ID %d already exists, skipping import...\n", utils.MustAtoi(record[0]))
            continue // Skip this task if it already exists
        }

        _, err = db.Exec(`INSERT INTO tasks (id, title, description, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
            utils.MustAtoi(record[0]), record[1], record[2], record[3], createdAt, updatedAt)
        if err != nil {
            return fmt.Errorf("error inserting task into database: %v", err)
        }
    }

    return nil
}
func init() {
    rootCmd.AddCommand(importCmd)
}
