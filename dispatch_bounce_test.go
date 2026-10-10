package main

import (
	"context"
	"testing"
)

type fakeEmailGateway struct {
	suppressed bool
	added      int
	sent       int
}

func (f *fakeEmailGateway) SuppressionCheck(context.Context, string) (bool, error) {
	return f.suppressed, nil
}
func (f *fakeEmailGateway) SuppressionAdd(context.Context, string) error { f.added++; return nil }
func (f *fakeEmailGateway) SendFollowUp(context.Context, string, string, string, string) (string, error) {
	f.sent++
	return "msg_42", nil
}

func TestDispatchFollowUpDecisions(t *testing.T) {
	order := WorkOrder{ID: "WO-42", CustomerEmail: "customer@example.com", DispatchStatus: "completed", PhotoURL: "https://photos.example.com/WO-42"}
	cases := []struct {
		name       string
		bounce     *Bounce
		suppressed bool
		action     string
		added      int
		sent       int
	}{
		{name: "hard bounce is suppressed", bounce: &Bounce{Email: order.CustomerEmail, Hard: true}, action: "suppressed", added: 1},
		{name: "existing suppression skips follow-up", suppressed: true, action: "skipped_suppressed"},
		{name: "deliverable customer receives photo follow-up", action: "sent", sent: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &fakeEmailGateway{suppressed: tc.suppressed}
			result, err := DispatchFollowUp(context.Background(), gateway, order, tc.bounce)
			if err != nil || result.Action != tc.action || gateway.added != tc.added || gateway.sent != tc.sent {
				t.Fatalf("result=%+v err=%v added=%d sent=%d", result, err, gateway.added, gateway.sent)
			}
		})
	}
}
