-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (gen_random_uuid(), NOW(), NOW(), $1)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET (updated_at, email) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: GetUsers :many
SELECT id, email, updated_at, created_at
FROM users
ORDER BY id;

-- name: GetUserByID :one
SELECT id, email, updated_at, created_at
FROM users
WHERE id = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;
