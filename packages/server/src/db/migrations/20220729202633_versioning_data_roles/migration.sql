/*
 This migration adds new columns and tables to support versioning of 
 routines and standards, storing data entered during a routine, 
 and changing the way roles work to be unique to organizaitons.
 
 Warnings:
 
 - You are about to drop the column `role` on the `organization_users` table. All the data in the column will be lost.
 - Added the required column `organizationId` to the `role` table without a default value. This is not possible if the table is not empty.
 - Added the required column `versionGroupId` to the `routine` table without a default value. This is not possible if the table is not empty.
 - Added the required column `versionGroupId` to the `standard` table without a default value. This is not possible if the table is not empty.
 
 */
-- Drops tag constraints for some reason. Prisma really wants to do this
-- for some reason
ALTER TABLE
    "organization_tags" DROP CONSTRAINT "organization_tags_tagTag_tagId_fkey";

ALTER TABLE
    "organization_tags" DROP CONSTRAINT "organization_tags_taggedId_tagTag_fkey";

ALTER TABLE
    "project_tags" DROP CONSTRAINT "project_tags_tagTag_tagId_fkey";

ALTER TABLE
    "project_tags" DROP CONSTRAINT "project_tags_taggedId_tagTag_fkey";

ALTER TABLE
    "routine_tags" DROP CONSTRAINT "routine_tags_tagTag_tagId_fkey";

ALTER TABLE
    "routine_tags" DROP CONSTRAINT "routine_tags_taggedId_tagTag_fkey";

ALTER TABLE
    "standard_tags" DROP CONSTRAINT "standard_tags_tagTag_tagId_fkey";

ALTER TABLE
    "standard_tags" DROP CONSTRAINT "standard_tags_taggedId_tagTag_fkey";

-- Change organization table rows
ALTER TABLE
    "organization"
ADD
    COLUMN "isPrivate" BOOLEAN NOT NULL DEFAULT false;

-- Change organization_users table. Now this is used to 
-- store members of an organization, without specifying 
-- their role.
ALTER TABLE
    "organization_users" DROP COLUMN "role";

-- Change project table rows
ALTER TABLE
    "project"
ADD
    COLUMN "isPrivate" BOOLEAN NOT NULL DEFAULT false;

-- Delete user_roles table, since we are completely changing 
-- how it works and don't need the old data anymore.
DROP TABLE "user_roles";

-- Delete role table, since we don't need the old data anyway
DROP TABLE "role";

-- Create new role table and translations table
CREATE TABLE "role" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "title" VARCHAR(128) NOT NULL,
    "organizationId" UUID NOT NULL,
    CONSTRAINT "role_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "role_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "role_translation_pkey" PRIMARY KEY ("id")
);

-- Create new user_roles table
CREATE TABLE "user_roles" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID NOT NULL,
    "roleId" UUID NOT NULL,
    CONSTRAINT "user_roles_pkey" PRIMARY KEY ("id")
);

-- Change routine table rows
ALTER TABLE
    "routine"
ADD
    COLUMN "isDeleted" BOOLEAN NOT NULL DEFAULT false,
ADD
    COLUMN "isPrivate" BOOLEAN NOT NULL DEFAULT false,
ADD
    COLUMN "versionGroupId" UUID;

-- Change run table rows
ALTER TABLE
    "run"
ADD
    COLUMN "isPrivate" BOOLEAN NOT NULL DEFAULT false;

-- Change standard table rows
ALTER TABLE
    "standard"
ADD
    COLUMN "isDeleted" BOOLEAN NOT NULL DEFAULT false,
ADD
    COLUMN "isPrivate" BOOLEAN NOT NULL DEFAULT false,
ADD
    COLUMN "versionGroupId" UUID;

-- Drop old role enum
DROP TYPE "MemberRole";

-- Create table to store run input data
CREATE TABLE "run_input" (
    "id" UUID NOT NULL,
    "data" VARCHAR(8192) NOT NULL,
    "runId" UUID NOT NULL,
    CONSTRAINT "run_input_pkey" PRIMARY KEY ("id")
);

-- Add tag constraint back in
ALTER TABLE
    "organization_tags"
ADD
    CONSTRAINT "organization_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE
    "project_tags"
ADD
    CONSTRAINT "project_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE
    "routine_tags"
ADD
    CONSTRAINT "routine_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE
    "standard_tags"
ADD
    CONSTRAINT "standard_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- Add foreign key constraint to run_input table
ALTER TABLE
    "run_input"
ADD
    CONSTRAINT "run_input_runId_fkey" FOREIGN KEY ("runId") REFERENCES "run"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Create foreign key constraint on role table
ALTER TABLE
    "role"
ADD
    CONSTRAINT "role_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization" ("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Create foreign key constraints on user_roles table
ALTER TABLE
    "user_roles"
ADD
    CONSTRAINT "user_roles_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user" ("id");

ALTER TABLE
    "user_roles"
ADD
    CONSTRAINT "user_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role" ("id");

-- Create unique index for a role's organizationId and title
CREATE UNIQUE INDEX "role_organizationId_title_key" ON "role" ("organizationId", "title");

-- Create unique index for a role's translation's language and roleId
CREATE UNIQUE INDEX "role_translation_language_roleId_key" ON "role_translation" ("language", "roleId");

-- Create unique index for a user_role's userId and roleId
CREATE UNIQUE INDEX "user_roles_userId_roleId_key" ON "user_roles" ("userId", "roleId");

-- Create foreign key for a role's translation's roleId
ALTER TABLE
    "role_translation"
ADD
    CONSTRAINT "role_translation_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role" ("id") ON DELETE CASCADE ON UPDATE CASCADE;