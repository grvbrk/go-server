-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
gen_random_uuid(), NOW(), NOW(), $1
)
RETURNING *;

-- name: DeleteUserByEmail :exec
DELETE FROM users
WHERE email = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;
