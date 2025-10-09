## Chirpy Server
**Chirpy Server** is a simple implementation of a REST API using the **Go** language. This project is intended as a basic example of how to build a RESTful API using `net/http`.

#### Feature

* CRUD endpoints for `users` and `chrips`.
* Supports `GET`, `POST`, `PATCH`, and `DELETE` operations.
* Response in JSON format.

#### Endpoints

* `GET /app/` →  a simple html serve.
* `GET /api/health` →  checking the health of API.
* `GET /api/users` →  Retrieve all users.
* `GET /api/users/{id}` →  Retrieve users by ID.
* `GET /api/chirps` →  Retrieve all chirps, filter chirp using `author_id=<user_id>` query param, and sort by asc (default) or desc by passing `sort=asc|desc` query param.
* `GET /api/chirps/{id}` →  Retrieve chirp by chrip ID.
* `POST /api/login` →  Login with email and password. Generate an accdess token (exp. 1 hours) and refresh token (exp. 60 days).
* `POST /api/users` →  Create a new user with a JSON request body (e.g., email, password).
* `POST /api/chirps` →  Create a new chirp with a JSON request body (e.g., body, user_id) and require a valid access token in Authorization Header.
* `POST /api/refresh` →  Refresh access token.
* `POST /api/revoke` →  Revoke refresh token.
* `POST /api/polka/webhooks` →  Upgrade user subscription.
* `PUT /api/users` →  Idempotent update user data.
* `PATCH /api/chirps/{id}` →  Update partial chrip data.
* `DELETE /api/users/{id}` →  Delete user by ID.
* `DELETE /api/chrips/{chirpID}` →  Delete chirp by ID.
* `GET /admin/metrics` →  Show the user metrics count.
* `POST /admin/reset` →  Reset the metrics count and delete all users.

#### Tech Stack

* [Go](https://pkg.go.dev/net/http) (`net/http`)
* [PostgreSQL](https://www.postgresql.org/)
* [Goose](https://pressly.github.io/goose/)
* [SQLC](https://docs.sqlc.dev/en/latest/index.html)

#### Run the App

```
go build -o <name> && ./<name>
```

#### Note
This project is intended as an example of basic REST API learning with Go, not for live production.
