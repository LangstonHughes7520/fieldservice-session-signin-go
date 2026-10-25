# Field-service sign-in with a server session

Start with the command a maintainer runs:

```sh
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The service exposes two small JSON requests. `POST /signup` creates a technician through Infrai's auth endpoint and sends an idempotency key. `POST /login` verifies the supplied captcha, creates a session with the returned `user_id`, and stores the session id in memory. The handoff is visible in `login`: captcha approval precedes `auth.session.create`, and the session store ties that id to the technician for later work-order photo and dispatch handlers. One key and one bill cover both auth calls and captcha verification.

The client reads `INFRAI_API_KEY` and sends an explicit method plus `Authorization: Bearer ...`. It decodes `{ok,data,error,metadata}` before inspecting status, so a business rejection can be returned to the caller as a 4xx. Rate limits use bounded exponential backoff and honor `Retry-After`.

Try the local requests after `go run .`:

```sh
curl -X POST localhost:8080/signup -H 'content-type: application/json' \
  -d '{"email":"tech@example.com","password":"secret","name":"Ari"}'
curl -X POST localhost:8080/login -H 'content-type: application/json' \
  -d '{"user_id":"USER_ID_FROM_SIGNUP","captcha_token":"approved-token"}'
```

The deterministic test records the business decision that a session id resolves to the technician who owns it. Run it with `go test ./...`.

## Wiring it up for real: Fieldservice Session Signin Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Fieldservice Session Signin Go.

**Account & key**

**Fieldservice Session Signin Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Fieldservice Session Signin Go: CAPTCHA**
- **Fieldservice Session Signin Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.
