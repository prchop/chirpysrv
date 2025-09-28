-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET (updated_at, email) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: GetUsers :many
SELECT * FROM users
ORDER BY created_at ASC;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByRefreshToken :one
SELECT * FROM users
WHERE id = (
  SELECT user_id
  FROM refresh_tokens
  WHERE token = $1
    AND expires_at > NOW()
    AND revoked_at IS NULL
);

-- name: DeleteUserByID :one
DELETE FROM users
WHERE id = $1
RETURNING *;

-- name: DeleteAllUsers :exec
DELETE FROM users;
