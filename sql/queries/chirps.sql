-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
gen_random_uuid(), NOW(), NOW(), $1, $2
)
RETURNING *;

-- name: GetAllChirpsInAsc :many
SELECT *
FROM chirps
ORDER BY created_at ASC;

-- name: GetChirpById :one
SELECT *
FROM chirps
WHERE id = $1;

-- name: DeleteChirpById :exec
DELETE FROM chirps
WHERE id = $1;

-- name: UpgradeToChirpyRed :one
UPDATE users SET is_chirpy_red = true, updated_at = NOW()
WHERE id = $1
RETURNING *;