-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  token,
  created_at,
  updated_at,
  user_id,
  expires_at,
  revoked_at
)
VALUES ($1, NOW(), NOW(), $2, $3, $4)
RETURNING *;

-- name: RevokeRefreshToken :one
UPDATE refresh_tokens SET (updated_at, revoked_at) = (NOW(), NOW())
WHERE token = $1
RETURNING *;
