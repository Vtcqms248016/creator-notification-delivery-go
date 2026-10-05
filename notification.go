package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Notification struct {
	ID        string `json:"id"`
	CreatorID string `json:"creator_id"`
	UserID    string `json:"user_id"`
	Kind      string `json:"kind"`
	Online    bool   `json:"online"`
	Message   string `json:"message"`
}

type Delivery struct {
	ID    string `json:"id"`
	Route string `json:"route"`
}

type sender interface {
	Publish(context.Context, string, string, any, string) error
	Queue(context.Context, any, string) error
}

func deliver(ctx context.Context, api sender, n Notification) (Delivery, error) {
	if strings.TrimSpace(n.ID) == "" || strings.TrimSpace(n.CreatorID) == "" || strings.TrimSpace(n.UserID) == "" || strings.TrimSpace(n.Message) == "" {
		return Delivery{}, errors.New("id, creator_id, user_id and message are required")
	}
	switch n.Kind {
	case "digital_asset", "subscriber_update", "content_processed":
	default:
		return Delivery{}, fmt.Errorf("unsupported kind: %s", n.Kind)
	}
	// The application owns this presence snapshot; offline events enter the durable queue.
	if !n.Online {
		if err := api.Queue(ctx, n, n.ID); err != nil {
			return Delivery{}, err
		}
		return Delivery{n.ID, "queued"}, nil
	}
	if err := api.Publish(ctx, "creator-"+n.CreatorID, n.Kind, n, n.ID); err != nil {
		return Delivery{}, err
	}
	return Delivery{n.ID, "realtime"}, nil
}
