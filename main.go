package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hamptonmoore/dublinbikes/client"
)

func main() {

	// Define the flags
	accountID := flag.String("account_id", "", "Account ID")
	accountEmail := flag.String("account_email", "", "Account Email")
	accountPassword := flag.String("account_password", "", "Account Password")

	// Parse the flags
	flag.Parse()

	// Check if environment variables are set and use them if flags are not provided
	if *accountID == "" {
		if envAccountID, exists := os.LookupEnv("ACCOUNT_ID"); exists {
			*accountID = envAccountID
		}
	}

	if *accountEmail == "" {
		if envAccountEmail, exists := os.LookupEnv("ACCOUNT_EMAIL"); exists {
			*accountEmail = envAccountEmail
		}
	}

	if *accountPassword == "" {
		if envAccountPassword, exists := os.LookupEnv("ACCOUNT_PASSWORD"); exists {
			*accountPassword = envAccountPassword
		}
	}

	client, err := client.NewDublinBikesClient(*accountID, *accountEmail, *accountPassword)
	if err != nil {
		fmt.Printf("Error initializing client: %v\n", err)
		return
	}

	trips, err := client.GetTrips()
	if err != nil {
		fmt.Printf("Error getting trips: %v\n", err)
		return
	}

	for i, trip := range trips {
		fmt.Printf("Trip %d:\n", i+1)
		fmt.Printf("\tStart: %s, Station: %d\n", trip.StartDateTime, trip.StartStation)
		fmt.Printf("\tEnd: %s, Station: %d\n", trip.EndDateTime, trip.EndStation)
		fmt.Printf("\tDuration: %d minutes\n", trip.Duration)
	}

}
