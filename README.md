## Chirpy Server
**Chirpy Server** is a simple implementation of a REST API using the **Go** language. This project is intended as a basic example of how to build a RESTful API using `net/http`.

#### Feature

* CRUD endpoints for `users` and `chrips`.
* Supports `GET`, `POST`, `PATCH`, and `DELETE` operations.
* Response in JSON format.

#### Endpoints

* `GET /app/` →  a simple interface to send JSON request body to `/api/users` endpoint.
* `GET /api/health` →  checking the health of API.
* `GET /api/users` →  Retrieve all users.
* `GET /api/chirps` →  Retrieve all chirps.
* `GET /api/users/{id}` →  Retrieve users by ID.
* `GET /api/chirps/{id}` →  Retrieve chirps by ID.
* `POST /api/users` →  Create a new user with a JSON request body (e.g., email).
* `POST /api/chirps` →  Create a new user with a JSON request body (e.g., body, user_id).
* `PATCH /api/users/{id}` →  Update partial user data.
* `PATCH /api/chirps/{id}` →  Update partial chrip data.
* `DELETE /api/users/{id}` →  Delete user by ID.
* `DELETE /api/chrips/{id}` →  Delete user by ID.
* `POST /admin/metrics` →  Show the user metrics count.
* `POST /admin/reset` →  Reset the metrics count and delete all users.

#### Tech Stack

* [Go](https://go.dev/doc/) (`net/http`)
* [PostgreSQL](https://www.postgresql.org/)
* [SQLC](https://docs.sqlc.dev/en/latest/index.html)

#### How to Run

```bash
make
```

or

```
go build -o <name> && ./<name>
```

#### Catatan
This project is intended as an example of basic REST API learning with Go, not for live production.
