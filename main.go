package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

func main() {
	to := os.Getenv("FIELD_SERVICE_CUSTOMER_EMAIL")
	if to == "" {
		log.Fatal("FIELD_SERVICE_CUSTOMER_EMAIL is required")
	}
	client, err := NewEmailClient()
	if err != nil {
		log.Fatal(err)
	}
	result, err := DispatchFollowUp(context.Background(), client, WorkOrder{
		ID: "WO-204", CustomerEmail: to, DispatchStatus: "technician completed the visit", PhotoURL: "https://photos.example.com/WO-204",
	}, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("action=%s message_id=%s\n", result.Action, result.MessageID)
}
