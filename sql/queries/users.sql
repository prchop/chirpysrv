-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (gen_random_uuid(), NOW(), NOW(), $1)
RETURNING *;

-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET (updated_at, email) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: UpdateChirp :one
UPDATE chirps SET (updated_at, body) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;
