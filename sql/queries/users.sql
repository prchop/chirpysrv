-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING id, created_at, updated_at, email;

-- name: UpdateUser :one
UPDATE users SET (updated_at, email) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: GetUsers :many
SELECT id, email, updated_at, created_at
FROM users
ORDER BY created_at ASC;

-- name: GetUserByID :one
SELECT id, email, updated_at, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: DeleteUserByID :one
DELETE FROM users
WHERE id = $1
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;
