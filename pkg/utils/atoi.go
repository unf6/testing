package utils

import (
	"fmt"
	"strconv"
)

// mustAtoi converts a string to an integer, returning 0 on error
func MustAtoi(s string) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		fmt.Printf("Error converting string to int: %v\n", err)
		return 0
	}
	return val
}