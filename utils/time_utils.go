package utils

import (
	"fmt"
	"time"
)

// var dateStr = "2025-03-09T00:00:00Z"
func ISOToUnix(dateISO string) int64 {

	t, err := time.Parse(time.RFC3339, dateISO)
	if err != nil {
		fmt.Println(err)
		return -1
	}
	// Convert to Unix timestamp (seconds)
	unixTime := t.Unix()

	return unixTime
}
