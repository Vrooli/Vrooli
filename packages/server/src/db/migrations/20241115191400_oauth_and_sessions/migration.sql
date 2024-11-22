-- Create the user_auth table with default UUIDs
CREATE TABLE "user_auth" (
    "id" UUID NOT NULL DEFAULT uuid_generate_v4(),
    "user_id" UUID NOT NULL,
    "provider" TEXT NOT NULL,
    "provider_user_id" TEXT,
    "hashed_password" TEXT,
    "resetPasswordCode" VARCHAR(256),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "user_auth_pkey" PRIMARY KEY ("id")
);

-- Create the session table with default UUIDs
CREATE TABLE "session" (
    "id" UUID NOT NULL DEFAULT uuid_generate_v4(),
    "user_id" UUID NOT NULL,
    "auth_id" UUID NOT NULL,
    "refresh_token" VARCHAR(1024) NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expires_at" TIMESTAMPTZ(6) NOT NULL,
    "revoked" BOOLEAN NOT NULL DEFAULT false,
    "device_info" VARCHAR(1024),
    "ip_address" VARCHAR(45),

    CONSTRAINT "session_pkey" PRIMARY KEY ("id")
);

-- Create indexes for user_auth
CREATE UNIQUE INDEX "user_auth_resetPasswordCode_key" ON "user_auth"("resetPasswordCode");
CREATE INDEX "user_auth_user_id_idx" ON "user_auth"("user_id");
CREATE UNIQUE INDEX "user_auth_provider_provider_user_id_key" ON "user_auth"("provider", "provider_user_id");

-- Create indexes for session
CREATE UNIQUE INDEX "session_refresh_token_key" ON "session"("refresh_token");
CREATE INDEX "session_user_id_idx" ON "session"("user_id");
CREATE INDEX "session_auth_id_idx" ON "session"("auth_id");

-- Add foreign keys for user_auth
ALTER TABLE "user_auth" ADD CONSTRAINT "user_auth_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Add foreign keys for session
ALTER TABLE "session" ADD CONSTRAINT "session_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "session" ADD CONSTRAINT "session_auth_id_fkey" FOREIGN KEY ("auth_id") REFERENCES "user_auth"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- **Data Migration Step: Transfer existing passwords and reset codes to user_auth**
INSERT INTO "user_auth" ("id", "user_id", "provider", "hashed_password", "resetPasswordCode", "created_at", "updated_at")
SELECT
    uuid_generate_v4(),      -- Generate a new UUID for id
    "id",                    -- user_id from user table
    'password',              -- provider is 'password' for all existing passwords
    "password",              -- hashed_password from user table
    "resetPasswordCode",     -- resetPasswordCode from user table
    "created_at",            -- Preserve original creation date
    "updated_at"             -- Preserve original update date
FROM "user"
WHERE "password" IS NOT NULL OR "resetPasswordCode" IS NOT NULL;

-- Now that data is migrated, we can safely drop the old columns
-- DropIndex (if it exists)
DROP INDEX IF EXISTS "user_resetPasswordCode_key";

-- AlterTable: Drop columns from user table
ALTER TABLE "user"
    DROP COLUMN IF EXISTS "password",
    DROP COLUMN IF EXISTS "resetPasswordCode",
    DROP COLUMN IF EXISTS "sessionToken";
