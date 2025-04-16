CREATE TABLE IF NOT EXISTS public.users
(
    user_id bigint NOT NULL DEFAULT nextval('users_user_id_seq'::regclass),
    email text NOT NULL,
    password text NOT NULL,
    role bigint NOT NULL,
    user_uiid uuid NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed boolean NOT NULL DEFAULT false,
    deleted boolean NOT NULL DEFAULT false,
    clerk_id text NOT NULL DEFAULT 'id'::text,
    CONSTRAINT users_pkey PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS public.services
(
    sid bigint NOT NULL DEFAULT nextval('services_service_id_seq'::regclass),
    user_id bigint NOT NULL,
    service_uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    key_id bigint NOT NULL,
    name text NOT NULL,
    CONSTRAINT services_pkey PRIMARY KEY (sid),
    CONSTRAINT keys_services_key_id_fkey FOREIGN KEY (key_id)
        REFERENCES public.keys (key_id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
        NOT VALID,
    CONSTRAINT users_services_user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public.users (user_id) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
        NOT VALID
);

CREATE TABLE IF NOT EXISTS public.keys
(
    key_id bigint NOT NULL DEFAULT nextval('keys_key_id_seq'::regclass),
    key text COLLATE pg_catalog."default" NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cache boolean NOT NULL DEFAULT false,
    storage boolean NOT NULL DEFAULT false,
    expires_at bigint NOT NULL,
    id text COLLATE pg_catalog."default" NOT NULL,
    CONSTRAINT keys_pkey PRIMARY KEY (key_id)
);

CREATE TABLE IF NOT EXISTS public.storage
(
    str_id bigint NOT NULL DEFAULT nextval('storage_storage_id_seq'::regclass),
    service_id bigint NOT NULL,
    upload boolean NOT NULL DEFAULT false,
    download boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT storage_pkey PRIMARY KEY (str_id),
    CONSTRAINT services_storage_service_id_fkey FOREIGN KEY (service_id)
        REFERENCES public.services (sid) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
        NOT VALID
);

CREATE TABLE IF NOT EXISTS public.cache
(
    cch_id bigint NOT NULL DEFAULT nextval('cache_cch_id_seq'::regclass),
    service_id bigint NOT NULL,
    get boolean NOT NULL DEFAULT false,
    put boolean NOT NULL DEFAULT false,
    created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT cache_pkey PRIMARY KEY (cch_id),
    CONSTRAINT services_cache_cch_id_fkey FOREIGN KEY (service_id)
        REFERENCES public.services (sid) MATCH SIMPLE
        ON UPDATE CASCADE
        ON DELETE CASCADE
        NOT VALID
);