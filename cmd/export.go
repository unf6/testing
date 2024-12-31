package cmd

import (
	"strings"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/unf6/testing/pkg/database"
	"github.com/unf6/testing/models"
	"github.com/unf6/testing/pkg/utils"
)

// Tasks represents the structure of a task.


// exportCmd represents the export command
var exportCmd = &cobra.Command{
	Use:   "export [json|txt]",
	Short: "Export tasks from SQLite or CSV file",
	Long: `Export tasks to a specified file format (JSON or TXT).
If no arguments are provided, you will be prompted to select the format interactively.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prompt for data source
		sourcePrompt := promptui.Select{
			Label: "Select data source",
			Items: []string{"SQLite", "CSV"},
		}
		_, source, err := sourcePrompt.Run()
		if err != nil {
			fmt.Printf("Error during source selection: %v\n", err)
			return
		}

		var tasks []models.Task
		if source == "SQLite" {
			tasks = fetchTasksFromSQLite()
		} else {
			tasks = fetchTasksFromCSV()
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found to export.")
			return
		}

		// Determine export format
		var format string
		if len(args) > 0 {
			format = args[0]
			if format != "json" && format != "txt" {
				fmt.Println("Invalid format specified. Valid options are 'json' or 'txt'.")
				return
			}
		} else {
			formatPrompt := promptui.Select{
				Label: "Select export format",
				Items: []string{"JSON", "TXT"},
			}
			_, format, err = formatPrompt.Run()
			if err != nil {
				fmt.Printf("Error during format selection: %v\n", err)
				return
			}
			format = strings.ToLower(format)
		}

		// Prompt for file name
		filePrompt := promptui.Prompt{
			Label:   "Enter export file name (without extension)",
			Default: fmt.Sprintf("tasks_export_%s", time.Now().Format("20060102_150405")),
		}
		fileName, err := filePrompt.Run()
		if err != nil {
			fmt.Printf("Error during file name input: %v\n", err)
			return
		}

		// Export tasks
		switch format {
		case "json":
			exportToJSON(tasks, fileName+".json")
		case "txt":
			exportToTXT(tasks, fileName+".txt")
		default:
			fmt.Println("Invalid format selected.")
		}
	},
}

// fetchTasksFromSQLite fetches tasks from the SQLite database
func fetchTasksFromSQLite() []models.Task {
	db := database.GetDB()
	rows, err := db.Query("SELECT id, title, description, status, created_at, updated_at FROM tasks")
	if err != nil {
		fmt.Printf("Error querying SQLite database: %v\n", err)
		return nil
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var createdAt, updatedAt string
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &createdAt, &updatedAt)
		if err != nil {
			fmt.Printf("Error scanning SQLite row: %v\n", err)
			return nil
		}

		task.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		task.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		tasks = append(tasks, task)
	}
	return tasks
}

// fetchTasksFromCSV fetches tasks from the CSV file
func fetchTasksFromCSV() []models.Task {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "tasks-cli")
	csvFilePath := filepath.Join(configDir, "tasks.csv")

	file, err := os.Open(csvFilePath)
	if err != nil {
		fmt.Printf("Error opening CSV file: %v\n", err)
		return nil
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Error reading CSV file: %v\n", err)
		return nil
	}

	var tasks []models.Task
	for i, record := range records {
		if i == 0 {
			// Skip header row
			continue
		}

		createdAt, _ := time.Parse(time.RFC3339, record[4])
		updatedAt, _ := time.Parse(time.RFC3339, record[5])

		tasks = append(tasks, models.Task{
			ID:          utils.MustAtoi(record[0]),
			Title:       record[1],
			Description: record[2],
			Status:      record[3],
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
	return tasks
}

// exportToJSON exports tasks to a JSON file
func exportToJSON(tasks []models.Task, fileName string) {
	filePath := filepath.Join(".", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(tasks); err != nil {
		fmt.Printf("Error writing to JSON file: %v\n", err)
		return
	}

	fmt.Printf("Tasks exported successfully to %s\n", filePath)
}

// exportToTXT exports tasks to a TXT file
func exportToTXT(tasks []models.Task, fileName string) {
	filePath := filepath.Join(".", fileName)
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	for _, task := range tasks {
		line := fmt.Sprintf("ID: %d\nTitle: %s\nDescription: %s\nStatus: %s\nCreatedAt: %s\nUpdatedAt: %s\n\n",
			task.ID, task.Title, task.Description, task.Status,
			task.CreatedAt.Format(time.RFC3339), task.UpdatedAt.Format(time.RFC3339))
		_, err := file.WriteString(line)
		if err != nil {
			fmt.Printf("Error writing to TXT file: %v\n", err)
			return
		}
	}

	fmt.Printf("Tasks exported successfully to %s\n", filePath)
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
