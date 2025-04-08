

-- name: SignupUser :exec
INSERT INTO users (email, password, role)
VALUES ($1, $2, $3);


-- name: GetUserData :one
SELECT 
    users.user_id,
    users.email,
    users.role,
    users.user_uiid,
    users.created_at,
    users.confirmed,
    users.deleted
FROM users
WHERE users.user_id = $1;

-- name: GetServiceCountForUserID :one
SELECT
    COUNT(services.sid)
FROM services
WHERE services.user_id = $1
AND services.name = $2;


-- name: InsertKey :one
INSERT INTO keys (key, cache, storage, expires_at, id)
VALUES ($1, $2, $3, $4, $5)
RETURNING key_id, created_at;

-- name: UpdateKeyServicesConfirmation :one
UPDATE keys 
SET 
    cache = $2,
    storage = $3
WHERE key_id = $1
RETURNING cache, storage;



-- name: InsertNewService :one
INSERT INTO services (user_id, key_id, name)
VALUES ($1, $2, $3)
RETURNING service_uuid;


-- name: GetUserIDFromClerkID :one
SELECT 
    users.user_id
FROM users
WHERE users.clerk_id = $1;


-- name: GetServiceData :one
SELECT
    services.sid,
    services.user_id,
    services.key_id
FROM services
WHERE services.name = $1;



-- name: DeleteKey :exec
DELETE FROM keys
WHERE keys.key_id = $1;


-- name: DeleteService :exec
DELETE FROM services
WHERE services.sid = $1;





-- >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
-- CACHE QUERIES


-- name: GetUserDataFromAPIKey :one
SELECT
    keys.key_id,
    keys.created_at,
    keys.updated_at,
    keys.cache,
    keys.storage,

    users.user_id,
    users.role,
    users.user_uiid,
    users.confirmed
FROM keys
JOIN users ON keys.user_id = users.user_id 
WHERE keys.key = $1;
