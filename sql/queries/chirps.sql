-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING *;

-- name: UpdateChirp :one
UPDATE chirps SET (updated_at, body) = (NOW(), $1)
WHERE id = $2
RETURNING *;

-- name: GetChirps :many
SELECT id, body, updated_at, created_at, user_id
FROM chirps
ORDER BY id;

-- name: GetChirpByID :one
SELECT id, body, updated_at, created_at, user_id
FROM chirps
WHERE id = $1;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;
