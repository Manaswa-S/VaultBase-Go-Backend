

-- name: CheckUserExistence :one
SELECT
    COUNT(users.user_id)
FROM users
WHERE users.clerk_id = $1;

-- name: SignupUser :exec
INSERT INTO users (email, role, clerk_id)
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




-- name: GetAllProjects :many
SELECT
    services.service_uuid,
    services.created_at,
    services.name,

    keys.key,
    keys.created_at,
    keys.updated_at,
    keys.cache,
    keys.storage,
    keys.expires_at,
    keys.id
FROM services
LEFT JOIN keys ON services.key_id = keys.key_id
WHERE services.user_id = $1;



-- name: GetServiceIDFromAPIKey :one
SELECT
    services.sid
FROM services
LEFT JOIN keys ON services.key_id = keys.key_id
WHERE keys.key = $1;



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
JOIN services ON keys.key_id = services.key_id
JOIN users ON users.user_id = services.user_id
WHERE keys.key = $1;





-- >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
-- Data Analytics



-- -- name: GetLatestStorage :one
-- SELECT 
--     storage.last_up,
--     storage.up_count,
--     storage.last_down,
--     storage.down_count
-- FROM storage 
-- WHERE storage.service_id = $1
-- ORDER BY storage.last_up DESC
-- LIMIT 1;


-- name: InsertStorageData :exec
INSERT INTO storage (service_id, upload, download, created_at) 
VALUES ($1, $2, $3, $4);


-- name: GetAllStorageData :many
SELECT
    storage.upload,
    storage.download,
    storage.created_at
FROM storage
WHERE storage.service_id = $1;