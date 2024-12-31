/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
    "path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/unf6/testing/pkg/database"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Remove a task",
	Long:  `Remove a task by its ID`,
	Run: func(cmd *cobra.Command, args []string) {
		id, err := cmd.Flags().GetString("id")

		if err != nil {
			fmt.Printf("Failed to get id flag: %v", err)
			os.Exit(1)
		}
		prompt := promptui.Select{
			Label:     "Where would you like to delete from?",
			Items:     []string{"Database (sqlite)", "CSV File"},
			CursorPos: 0,
		}

		_, choice, promptErr := prompt.Run()

		if promptErr != nil {
			fmt.Printf("%s Error: %v\n", promptui.IconBad, promptErr)
			os.Exit(1)
		}

		parsedInt, parsedErr := strconv.Atoi(id)
		if parsedErr != nil {
			fmt.Printf("%s ID needs to be an interger %v\n", promptui.IconBad, parsedErr)
			os.Exit(1)
		}

		switch choice {
		case "CSV File":
			deleteFromCSVFile(parsedInt)
		default:
			deleteFromDB(parsedInt)
		}
	},
}

func init() {
	deleteCmd.Flags().StringP("id", "d", "", "Remove a task by its ID")
	deleteCmd.MarkFlagRequired("id")

	deleteCmd.RegisterFlagCompletionFunc("id", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"1"}, cobra.ShellCompDirectiveNoFileComp
	})
	rootCmd.AddCommand(deleteCmd)
}

func deleteFromDB(id int) {
	db := database.GetDB()

	query := `DELETE FROM tasks WHERE id=?`

	result, resultsErr := db.Exec(query, id)

	if resultsErr != nil {
		fmt.Printf("%v Failed to delete task from Database (sqlite): %v", promptui.IconBad, resultsErr)
		os.Exit(1)
	}

	// Check if any rows were affected (i.e., a record was deleted)
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("%v Error getting rows affected: %v\n", promptui.IconBad, err)
		os.Exit(1)
	}

	if rowsAffected > 0 {
		fmt.Printf("%v Successfully deleted task with ID %d\n", promptui.IconGood, id)
		fmt.Printf("%v Database (sqlite) Data updated\n", promptui.IconGood)
	} else {
		fmt.Printf("%v No task found with ID %d to delete\n", promptui.IconGood, id)
	}
}

func deleteFromCSVFile(id int) {

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

	// Open file with read and write permissions
	file, fileErr := os.OpenFile(csvFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if fileErr != nil {
		fmt.Printf("%v Failed to create/open CSV file: %v", promptui.IconBad, fileErr)
		os.Exit(1)
	}
	defer file.Close()

	// Read CSV File
	reader := csv.NewReader(file)
	records, errRecords := reader.ReadAll()
	if errRecords != nil {
		fmt.Printf("%v Error loading CSV File records: %v", promptui.IconBad, errRecords)
		os.Exit(1)
	}

	data := getDataFromCSVFile(records)

	if len(data) <= 0 {
		fmt.Printf("%v No tasks found in the CSV file\n", promptui.IconGood)
		return
	}

	// Find and remove the task with matching ID
	newCSVData := []DBTask{}
	found := false
	for _, content := range data {
		if content.ID == id {
			found = true
			continue // Skip this record
		}
		newCSVData = append(newCSVData, content)
	}

	if !found {
		fmt.Printf("%v No task found with ID %d to delete.\n", promptui.IconGood, id)
		return
	}

	// Convert tasks back to CSV records
	var newRecords [][]string
	// Add headers
	newRecords = append(newRecords, []string{"ID", "TITLE", "DESCRIPTION", "STATUS", "CREATED AT", "UPDATED AT"})
	for _, task := range newCSVData {
		record := []string{
			strconv.Itoa(task.ID),
			task.Title,
			task.Description,
			task.Status,
			task.CreatedAt,
			task.UpdatedAt,
		}
		newRecords = append(newRecords, record)
	}

	// Truncate the file and write from beginning
	if err := file.Truncate(0); err != nil {
		fmt.Printf("%v Error truncating file: %v", promptui.IconBad, err)
		return
	}
	if _, err := file.Seek(0, 0); err != nil {
		fmt.Printf("%v Error seeking file: %v", promptui.IconBad, err)
		return
	}

	// Write the updated records
	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.WriteAll(newRecords); err != nil {
		fmt.Printf("%v Error writing to CSV: %v", promptui.IconBad, err)
		return
	}

	fmt.Printf("%v Task with ID %d deleted successfully.\n", promptui.IconGood, id)
	fmt.Printf("%v CSV Data updated\n", promptui.IconGood)
}