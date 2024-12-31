/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/mergestat/timediff"
	"github.com/spf13/cobra"
	"github.com/unf6/testing/pkg/database"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `Display all tasks with their titles, descriptions, and status`,
	Run: func(cmd *cobra.Command, args []string) {

		prompt := promptui.Select{
			Label:     "Which database should we list the data from?",
			Items:     []string{"Database (sqlite)", "CSV File"},
			CursorPos: 0,
		}

		_, listChoice, promptErr := prompt.Run()

		if promptErr != nil {
			fmt.Printf("%s Error: %v\n", promptui.IconBad, promptErr)
			os.Exit(1)
		}

		format, formatErr := cmd.Flags().GetString("format")

		if formatErr != nil {
			fmt.Printf("Failed to get format flag: %v", formatErr)
			os.Exit(1)
		}

		switch listChoice {
		case "CSV File":
			listFromCSVFile(format)
		default:
			listFromDatabase(format)
		}
	},
}

func init() {
	listCmd.Flags().StringP("format", "f", "table", "Output format: table, json")
	rootCmd.AddCommand(listCmd)
}

func listFromDatabase(format string) {
	// DB Connect
	db := database.GetDB()

	listAllQuery := `
		SELECT * FROM tasks;
	`
	rows, listAllErr := db.Query(listAllQuery)

	if listAllErr != nil {
		fmt.Printf("Failed to fetch all tasks: %v", listAllErr)
		os.Exit(1)
	}
	defer rows.Close()

	data := getRowData(rows)
	switch format {
	case "json":
		formatInJSON(data)
	default:
		formatInTable(data)
	}
}

func listFromCSVFile(format string) {

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

	// Open CSV File
	file, fileErr := os.OpenFile(csvFilePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)

	if fileErr != nil {
		fmt.Printf("Failed to open CSV File: %v", fileErr)
	}
	defer file.Close()

	// Read CSV File
	reader := csv.NewReader(file)
	records, errRecords := reader.ReadAll()

	if errRecords != nil {
		fmt.Printf("Error loading CSV File records: %v", errRecords)
		os.Exit(1)
	}

	data := getDataFromCSVFile(records)
	switch format {
	case "json":
		formatInJSON(data)
	default:
		formatInTable(data)
	}
}

func formatInTable(data []DBTask) {
	w := tabwriter.NewWriter(os.Stdout, 1, 0, 2, ' ', 0)
	// Headers
	fmt.Fprintln(w, "ID\tTITLE\tDESCRIPTION\tSTATUS\tCREATED AT\tUPDATED AT")

	// Separator line using dashes, adjusted to match column widths
	fmt.Fprintln(w, strings.Repeat("-", 3)+"\t"+
		strings.Repeat("-", 20)+"\t"+
		strings.Repeat("-", 30)+"\t"+
		strings.Repeat("-", 19)+"\t"+
		strings.Repeat("-", 12)+"\t"+
		strings.Repeat("-", 12))

	for _, task := range data {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			task.ID,
			task.Title,
			task.Description,
			task.Status,
			task.CreatedAt,
			task.UpdatedAt,
		)
	}
	w.Flush()
}

func formatInJSON(data []DBTask) {
	jsonData, err := json.MarshalIndent(data, "", " ")

	if err != nil {
		fmt.Printf("Failed to change data to JSON: %v", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

type DBTask struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func getRowData(rows *sql.Rows) []DBTask {
	tasks := make([]DBTask, 0)
	for rows.Next() {
		var id int
		var title, description, status, created_at, updated_at string

		err := rows.Scan(&id, &title, &description, &status, &created_at, &updated_at)
		if err != nil {
			fmt.Printf("Error scanning row: %v", err)
			os.Exit(1)
		}

		parsedCreatedDate, err := time.Parse("2006-01-02T15:04:05Z", created_at)
		if err != nil {
			fmt.Printf("Error parsing created_at: %v", err)
			os.Exit(1)
		}

		parsedUpdatedDate, err := time.Parse("2006-01-02T15:04:05Z", updated_at)
		if err != nil {
			fmt.Printf("Error parsing updated_at: %v", err)
			os.Exit(1)
		}

		if len(title) > 20 {
			title = title[:17] + "..."
		}
		if len(description) > 30 {
			description = description[:27] + "..."
		}

		task := DBTask{
			ID:          id,
			Title:       title,
			Description: description,
			Status:      status,
			CreatedAt:   timediff.TimeDiff(parsedCreatedDate),
			UpdatedAt:   timediff.TimeDiff(parsedUpdatedDate),
		}

		tasks = append(tasks, task)
	}
	return tasks
}

func getDataFromCSVFile(csvData [][]string) []DBTask {
	tasks := make([]DBTask, 0)
	// skip headers
	restOfContent := csvData[1:]
	for _, content := range restOfContent {
		if len(content) >= 6 {
			var id int

			id, errInnerId := strconv.Atoi(content[0])
			if errInnerId != nil {
				fmt.Printf("Failed to parse the ID: %v", errInnerId)
				break
			}
			parsedCreatedDate, err := time.Parse("2006-01-02 15:04:05.999999999 +0000 UTC", content[4])
			if err != nil {
				fmt.Printf("Error parsing created_at: %v", err)
				os.Exit(1)
			}

			parsedUpdatedDate, err := time.Parse("2006-01-02 15:04:05.999999999 +0000 UTC", content[5])
			if err != nil {
				fmt.Printf("Error parsing updated_at: %v", err)
				os.Exit(1)
			}
			task := DBTask{
				ID:          id,
				Title:       content[1],
				Description: content[2],
				Status:      content[3],
				CreatedAt:   timediff.TimeDiff(parsedCreatedDate),
				UpdatedAt:   timediff.TimeDiff(parsedUpdatedDate),
			}

			tasks = append(tasks, task)
		}
	}

	return tasks
}