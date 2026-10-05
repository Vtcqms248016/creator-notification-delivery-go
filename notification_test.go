package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recorder struct{ published, queued int }

func (r *recorder) Publish(_ context.Context, _, _ string, _ any, _ string) error {
	r.published++
	return nil
}
func (r *recorder) Queue(_ context.Context, _ any, _ string) error {
	r.queued++
	return nil
}

func TestDeliveryRoute(t *testing.T) {
	for _, tc := range []struct {
		name, kind, want string
		online           bool
		publish, queue   int
	}{
		{"asset online", "digital_asset", "realtime", true, 1, 0},
		{"subscriber offline", "subscriber_update", "queued", false, 0, 1},
		{"processing offline", "content_processed", "queued", false, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &recorder{}
			got, err := deliver(context.Background(), r, Notification{ID: "event-42", CreatorID: "c7", UserID: "u9", Kind: tc.kind, Online: tc.online, Message: "ready"})
			if err != nil || got.Route != tc.want || r.published != tc.publish || r.queued != tc.queue {
				t.Fatalf("delivery = %+v, published=%d queued=%d err=%v", got, r.published, r.queued, err)
			}
		})
	}
}

func TestEnvelopeBeforeHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Idempotency-Key") != "event-42" {
			t.Errorf("unexpected request method or idempotency key")
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"ok":false,"error":{"code":"INVALID_ARGUMENT","message":"invalid channel"}}`))
	}))
	defer server.Close()
	api := &infraiClient{baseURL: server.URL, key: "test-key", http: server.Client()}
	err := api.Publish(context.Background(), "creator-c7", "digital_asset", "ready", "event-42")
	var remote *apiError
	if !errors.As(err, &remote) || remote.Code != "INVALID_ARGUMENT" {
		t.Fatalf("want decoded business error, got %v", err)
	}
}
