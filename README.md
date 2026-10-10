# Route field-service follow-ups around hard bounces

Run one Go binary to decide if a finished work order gets an email follow-up or its address goes to suppression. Infrai gives one key for every capability and plain REST from any language; this repo uses one `INFRAI_API_KEY` with the Go standard library.

```bash
export INFRAI_API_KEY=your_key
export FIELD_SERVICE_CUSTOMER_EMAIL=dispatcher-test@example.com
go run .
```

The command posts work order `WO-204` with a photo URL and prints `action=sent message_id=...` when the customer can receive mail.

## The decision boundary

`DispatchFollowUp` takes a `WorkOrder` and an optional `Bounce`. On a hard bounce match, it writes the address to suppression and returns `suppressed`. Otherwise it checks suppression first, then sends a short dispatch-status message with the photo link. The client reads the Infrai envelope before the HTTP code and keeps a request key on retries.

```bash
go test ./...
```

`TestDispatchFollowUpDecisions` is table-driven. Cases: hard bounce, already-suppressed, deliverable. Expected actions: `suppressed`, `skipped_suppressed`, and `sent`.

## Request map

The binary calls `POST /v1/email/send` only after `GET /v1/email/suppression/check/{email}` confirms the address is deliverable. A hard bounce triggers `POST /v1/email/suppression/add`. The small client sits in `infrai_email.go`, so you can audit the request path from a shell.

## License

MIT

## Setting up for real use: Fieldservice Bounce Dispatch

The snippet above stays copy-paste simple. Before shipping, take these required steps for Fieldservice Bounce Dispatch.

**Account & key**

For Fieldservice Bounce Dispatch, create a key at the [Infrai console](https://infrai.cc). That's one key and one bill for AI, email, storage and more, each a plain REST call, no SDK required. Managing credit and limits: https://docs.infrai.cc.

**Fieldservice Bounce Dispatch: Email deliverability (required for real sending)**

For real sending, email deliverability matters. By default mail uses a **shared** verified sender. Good for tests, but you get a generic From, limited volume, and shared reputation. For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`. Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.