package main

import (
	"context"
	"fmt"
)

type WorkOrder struct {
	ID             string
	CustomerEmail  string
	DispatchStatus string
	PhotoURL       string
}

type Bounce struct {
	Email string
	Hard  bool
}

type FollowUpResult struct {
	Action    string
	MessageID string
}

type EmailGateway interface {
	SuppressionCheck(context.Context, string) (bool, error)
	SuppressionAdd(context.Context, string) error
	SendFollowUp(context.Context, string, string, string, string) (string, error)
}

// DispatchFollowUp keeps photo evidence and the dispatch state in the message body.
func DispatchFollowUp(ctx context.Context, email EmailGateway, order WorkOrder, bounce *Bounce) (FollowUpResult, error) {
	if bounce != nil && bounce.Hard && bounce.Email == order.CustomerEmail {
		if err := email.SuppressionAdd(ctx, order.CustomerEmail); err != nil {
			return FollowUpResult{}, err
		}
		return FollowUpResult{Action: "suppressed"}, nil
	}

	suppressed, err := email.SuppressionCheck(ctx, order.CustomerEmail)
	if err != nil {
		return FollowUpResult{}, err
	}
	if suppressed {
		return FollowUpResult{Action: "skipped_suppressed"}, nil
	}

	subject := fmt.Sprintf("Work order %s: %s", order.ID, order.DispatchStatus)
	html := fmt.Sprintf("<p>Dispatch status: %s</p><p>Photo: <a href=\"%s\">open work-order photo</a></p>", order.DispatchStatus, order.PhotoURL)
	messageID, err := email.SendFollowUp(ctx, order.CustomerEmail, subject, html, order.ID)
	if err != nil {
		return FollowUpResult{}, err
	}
	return FollowUpResult{Action: "sent", MessageID: messageID}, nil
}
