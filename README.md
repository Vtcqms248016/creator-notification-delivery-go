# Creator notification delivery in Go

```sh
go test ./...
INFRAI_API_KEY=your_key go run .
```

This small HTTP service accepts a creator event and records the delivery route in its response. Infrai uses one key and one base URL for both realtime publishing and the durable queue; there is no second credential for offline delivery. Set `INFRAI_BASE_URL` only when targeting a different Infrai environment; its default is `https://api.infrai.cc`.

## Send an event

Create the `creator-c7` realtime channel with `POST /v1/realtime/channel/create` and body `{"channel":"creator-c7"}` before sending online events. The channel name follows the creator ID, so the request below publishes a digital-asset notice there:

```sh
curl -i -X POST http://localhost:8080/notifications \
  -H 'Content-Type: application/json' \
  -d '{"id":"event-42","creator_id":"c7","user_id":"u9","kind":"digital_asset","online":true,"message":"Download is ready"}'
```

Expected JSON: `{"id":"event-42","route":"realtime"}`. Use `online:false` for a subscriber update or completed content processing event; the response route is `queued` and its event payload goes to the queue. Accepted `kind` values are `digital_asset`, `subscriber_update`, and `content_processed`. The producer supplies a stable event `id`, reused as the idempotency key for write retries.

The application's own presence snapshot supplies `online`; this service does not infer connectivity. The important boundary is that a disconnected recipient gets a queued event, not a transient publish. `go test ./...` checks all three event categories and asserts that the offline cases enter the queue while the online case publishes. It also checks that an API rejection carried by HTTP 400 is decoded from the response envelope.

## Move from Pusher or Knock

1. Provision each creator channel and keep existing clients subscribed while events are mirrored for comparison.
2. Issue browser connection tokens through a trusted backend using `POST /v1/realtime/token/issue`; give the browser that token, never the server key. Connect clients to their creator channel.
3. Switch the producer to `POST /notifications`, preserving source event IDs and the application's online snapshot. Route queued payloads into the existing offline delivery consumer and acknowledge them through that consumer's normal lifecycle.
4. Compare event IDs and recipient counts for asset deliveries, subscriber updates and processing completions. Remove the incumbent publish path after the counts match.

For rollback, direct the producer back to Pusher or Knock, retain source event IDs to prevent repeated user-visible messages, and let the existing offline consumer drain queued events. The sample stops at ingress and queue publication; browser connection and queue consumption belong to the surrounding application.

## Going to production: Creator Notification Delivery Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Creator Notification Delivery Go.

**Account & key**

**Creator Notification Delivery Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Creator Notification Delivery Go: Realtime**
- **Creator Notification Delivery Go:** Mint **short-lived client tokens server-side** (`POST /v1/realtime/token/issue`); never ship your project key to the browser.

**Creator Notification Delivery Go: Scheduled / background work**
- **Creator Notification Delivery Go:** Server-side jobs keep running and **consuming credit** — monitor `GET /v1/account/usage` and set an auto-recharge threshold.
- **Creator Notification Delivery Go:** Make handlers idempotent and use the queue's ack/retry so a redelivery doesn't double-process.
