CREATE EXTENSION IF NOT EXISTS citext;

-- CreateEnum
CREATE TYPE "AccountStatus" AS ENUM ('Deleted', 'Unlocked', 'SoftLocked', 'HardLocked');

-- CreateEnum
CREATE TYPE "MemberRole" AS ENUM ('Admin', 'Member', 'Owner');

-- CreateEnum
CREATE TYPE "NodeType" AS ENUM ('End', 'Redirect', 'RoutineList', 'Start');

-- CreateEnum
CREATE TYPE "ResourceUsedFor" AS ENUM ('Community', 'Context', 'Developer', 'Donation', 'ExternalService', 'Feed', 'Install', 'Learning', 'Notes', 'OfficialWebsite', 'Proposal', 'Related', 'Researching', 'Scheduling', 'Social', 'Tutorial');

-- CreateEnum
CREATE TYPE "ResourceListUsedFor" AS ENUM ('Custom', 'Display', 'Learn', 'Research', 'Develop');

-- CreateEnum
CREATE TYPE "RunStatus" AS ENUM ('Scheduled', 'InProgress', 'Completed', 'Failed', 'Cancelled');

-- CreateEnum
CREATE TYPE "RunStepStatus" AS ENUM ('InProgress', 'Completed', 'Skipped');

-- CreateTable
CREATE TABLE "comment" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "stars" INTEGER NOT NULL DEFAULT 0,
    "score" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "comment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment_translation" (
    "id" UUID NOT NULL,
    "text" VARCHAR(2048) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "commentId" UUID NOT NULL,

    CONSTRAINT "comment_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "email" (
    "id" UUID NOT NULL,
    "emailAddress" CITEXT NOT NULL,
    "receivesAccountUpdates" BOOLEAN NOT NULL DEFAULT true,
    "receivesBusinessUpdates" BOOLEAN NOT NULL DEFAULT true,
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(256),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "userId" UUID,

    CONSTRAINT "email_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "handle" (
    "id" UUID NOT NULL,
    "handle" VARCHAR(16),
    "walletId" UUID,

    CONSTRAINT "handle_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "columnIndex" INTEGER,
    "rowIndex" INTEGER,
    "type" "NodeType" NOT NULL,
    "routineId" UUID NOT NULL,

    CONSTRAINT "node_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "title" VARCHAR(128) NOT NULL DEFAULT E'Name Me',
    "language" VARCHAR(3) NOT NULL,
    "nodeId" UUID NOT NULL,

    CONSTRAINT "node_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_end" (
    "id" UUID NOT NULL,
    "wasSuccessful" BOOLEAN NOT NULL DEFAULT true,
    "nodeId" UUID NOT NULL,

    CONSTRAINT "node_end_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_link" (
    "id" UUID NOT NULL,
    "fromId" UUID NOT NULL,
    "routineId" UUID NOT NULL,
    "toId" UUID NOT NULL,
    "operation" VARCHAR(512),

    CONSTRAINT "node_link_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_link_when" (
    "id" UUID NOT NULL,
    "linkId" UUID NOT NULL,
    "condition" VARCHAR(8192) NOT NULL,

    CONSTRAINT "node_link_when_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_link_when_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "title" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "whenId" UUID NOT NULL,

    CONSTRAINT "node_link_when_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_loop" (
    "id" UUID NOT NULL,
    "loops" INTEGER DEFAULT 1,
    "maxLoops" INTEGER DEFAULT 1,
    "operation" VARCHAR(512),
    "nodeId" UUID NOT NULL,

    CONSTRAINT "node_loop_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_loop_while" (
    "id" UUID NOT NULL,
    "loopId" UUID NOT NULL,
    "condition" VARCHAR(8192) NOT NULL,

    CONSTRAINT "node_loop_while_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_loop_while_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "title" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "whileId" UUID NOT NULL,

    CONSTRAINT "node_loop_while_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_routine_list" (
    "id" UUID NOT NULL,
    "isOrdered" BOOLEAN NOT NULL DEFAULT false,
    "isOptional" BOOLEAN NOT NULL DEFAULT false,
    "nodeId" UUID NOT NULL,

    CONSTRAINT "node_routine_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_routine_list_item" (
    "id" UUID NOT NULL,
    "index" INTEGER DEFAULT 0,
    "isOptional" BOOLEAN NOT NULL DEFAULT false,
    "listId" UUID NOT NULL,
    "routineId" UUID NOT NULL,

    CONSTRAINT "node_routine_list_item_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_routine_list_item_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "title" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "itemId" UUID NOT NULL,

    CONSTRAINT "node_routine_list_item_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "handle" VARCHAR(16),
    "isOpenToNewMembers" BOOLEAN NOT NULL DEFAULT false,
    "stars" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "organization_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization_translation" (
    "id" UUID NOT NULL,
    "bio" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "organizationId" UUID NOT NULL,

    CONSTRAINT "organization_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "organization_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization_users" (
    "id" UUID NOT NULL,
    "organizationId" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "role" "MemberRole" NOT NULL DEFAULT E'Member',

    CONSTRAINT "organization_users_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "stars" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "createdByUserId" UUID,
    "createdByOrganizationId" UUID,
    "handle" VARCHAR(16),
    "userId" UUID,
    "organizationId" UUID,
    "parentId" UUID,

    CONSTRAINT "project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "projectId" UUID NOT NULL,

    CONSTRAINT "project_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "project_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report" (
    "id" UUID NOT NULL,
    "reason" VARCHAR(128) NOT NULL,
    "details" VARCHAR(1024),
    "language" VARCHAR(3) NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "fromId" UUID NOT NULL,
    "commentId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "tagId" UUID,
    "userId" UUID,

    CONSTRAINT "report_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "index" INTEGER DEFAULT 0,
    "link" VARCHAR(1024) NOT NULL,
    "usedFor" "ResourceUsedFor" NOT NULL DEFAULT E'Context',
    "listId" UUID NOT NULL,

    CONSTRAINT "resource_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "title" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "resourceId" UUID NOT NULL,

    CONSTRAINT "resource_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_list" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "index" INTEGER DEFAULT 0,
    "usedFor" "ResourceListUsedFor" NOT NULL DEFAULT E'Display',
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "userId" UUID,

    CONSTRAINT "resource_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_list_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(512),
    "title" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "listId" UUID NOT NULL,

    CONSTRAINT "resource_list_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "role" (
    "id" UUID NOT NULL,
    "title" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "role_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 1,
    "isAutomatable" BOOLEAN NOT NULL DEFAULT false,
    "isComplete" BOOLEAN NOT NULL DEFAULT true,
    "isInternal" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "simplicity" INTEGER NOT NULL DEFAULT 1,
    "stars" INTEGER NOT NULL DEFAULT 0,
    "timesStarted" INTEGER NOT NULL DEFAULT 0,
    "timesCompleted" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "createdByUserId" UUID,
    "createdByOrganizationId" UUID,
    "organizationId" UUID,
    "parentId" UUID,
    "projectId" UUID,
    "userId" UUID,
    "version" VARCHAR(16) NOT NULL DEFAULT E'1.0.0',

    CONSTRAINT "routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "instructions" VARCHAR(8192) NOT NULL,
    "title" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "routineId" UUID NOT NULL,

    CONSTRAINT "routine_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_input" (
    "id" UUID NOT NULL,
    "isRequired" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128),
    "routineId" UUID NOT NULL,
    "standardId" UUID,

    CONSTRAINT "routine_input_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_input_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "routineInputId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "routine_input_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_output" (
    "id" UUID NOT NULL,
    "name" VARCHAR(128),
    "routineId" UUID NOT NULL,
    "standardId" UUID,

    CONSTRAINT "routine_output_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_output_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "routineOutputId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "routine_output_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "routine_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "pickups" INTEGER NOT NULL DEFAULT 1,
    "timeStarted" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "timeCompleted" TIMESTAMPTZ(6),
    "title" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT E'Scheduled',
    "version" VARCHAR(16) NOT NULL,
    "routineId" UUID,
    "userId" UUID NOT NULL,

    CONSTRAINT "run_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_step" (
    "id" UUID NOT NULL,
    "order" INTEGER NOT NULL,
    "pickups" INTEGER NOT NULL DEFAULT 1,
    "runId" UUID NOT NULL,
    "nodeId" UUID NOT NULL,
    "timeStarted" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "timeCompleted" TIMESTAMPTZ(6),
    "step" INTEGER[],
    "status" "RunStepStatus" NOT NULL DEFAULT E'InProgress',
    "title" VARCHAR(128) NOT NULL,

    CONSTRAINT "run_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "default" VARCHAR(1024),
    "name" VARCHAR(128) NOT NULL,
    "score" INTEGER NOT NULL DEFAULT 0,
    "stars" INTEGER NOT NULL DEFAULT 0,
    "type" TEXT NOT NULL,
    "props" VARCHAR(8192) NOT NULL,
    "yup" VARCHAR(8192),
    "version" VARCHAR(16) NOT NULL DEFAULT E'1.0.0',
    "views" INTEGER NOT NULL DEFAULT 0,
    "createdByUserId" UUID,
    "createdByOrganizationId" UUID,
    "isFile" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "standard_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "standardId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "standard_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "standard_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "star" (
    "id" UUID NOT NULL,
    "byId" UUID NOT NULL,
    "commentId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "tagId" UUID,
    "userId" UUID,

    CONSTRAINT "star_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tag" VARCHAR(128) NOT NULL,
    "createdByUserId" UUID,
    "stars" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "tag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "tagId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "tag_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "confirmationCode" VARCHAR(256),
    "confirmationCodeDate" TIMESTAMPTZ(6),
    "lastExport" TIMESTAMPTZ(6),
    "lastLoginAttempt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastResetPasswordReqestAttempt" TIMESTAMPTZ(6),
    "logInAttempts" INTEGER NOT NULL DEFAULT 0,
    "lastSessionVerified" TIMESTAMPTZ(6),
    "numExports" INTEGER NOT NULL DEFAULT 0,
    "password" VARCHAR(256),
    "resetPasswordCode" VARCHAR(256),
    "sessionToken" VARCHAR(1024),
    "name" VARCHAR(128) NOT NULL,
    "theme" VARCHAR(255) NOT NULL DEFAULT E'light',
    "handle" VARCHAR(16),
    "stars" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "status" "AccountStatus" NOT NULL DEFAULT E'Unlocked',

    CONSTRAINT "user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_translation" (
    "id" UUID NOT NULL,
    "bio" VARCHAR(2048),
    "userId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "user_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_language" (
    "id" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "user_language_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_roles" (
    "id" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "user_roles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_tag_hidden" (
    "id" UUID NOT NULL,
    "isBlur" BOOLEAN NOT NULL DEFAULT true,
    "userId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "user_tag_hidden_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "view" (
    "id" UUID NOT NULL,
    "lastViewed" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "title" VARCHAR(128) NOT NULL,
    "byId" UUID NOT NULL,
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "userId" UUID,

    CONSTRAINT "view_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "vote" (
    "id" UUID NOT NULL,
    "isUpvote" BOOLEAN NOT NULL DEFAULT true,
    "byId" UUID NOT NULL,
    "commentId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,

    CONSTRAINT "vote_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "wallet" (
    "id" UUID NOT NULL,
    "stakingAddress" VARCHAR(128) NOT NULL,
    "publicAddress" VARCHAR(128),
    "name" VARCHAR(128),
    "nonce" VARCHAR(8092),
    "nonceCreationTime" TIMESTAMPTZ(6),
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "wasReported" BOOLEAN NOT NULL DEFAULT false,
    "userId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "wallet_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "comment_translation_commentId_language_key" ON "comment_translation"("commentId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "email_emailAddress_key" ON "email"("emailAddress");

-- CreateIndex
CREATE UNIQUE INDEX "email_verificationCode_key" ON "email"("verificationCode");

-- CreateIndex
CREATE UNIQUE INDEX "handle_handle_key" ON "handle"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "node_translation_nodeId_language_key" ON "node_translation"("nodeId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "node_end_nodeId_key" ON "node_end"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_link_when_translation_whenId_language_key" ON "node_link_when_translation"("whenId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "node_loop_nodeId_key" ON "node_loop"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_nodeId_key" ON "node_routine_list"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_item_listId_routineId_key" ON "node_routine_list_item"("listId", "routineId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_item_translation_itemId_language_key" ON "node_routine_list_item_translation"("itemId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "organization_handle_key" ON "organization"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "organization_translation_organizationId_language_key" ON "organization_translation"("organizationId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "organization_tags_taggedId_tagId_key" ON "organization_tags"("taggedId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "organization_users_organizationId_userId_key" ON "organization_users"("organizationId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "project_handle_key" ON "project"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "project_translation_projectId_language_key" ON "project_translation"("projectId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "project_tags_taggedId_tagId_key" ON "project_tags"("taggedId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "resource_translation_resourceId_language_key" ON "resource_translation"("resourceId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "resource_list_translation_listId_language_key" ON "resource_list_translation"("listId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "role_title_key" ON "role"("title");

-- CreateIndex
CREATE UNIQUE INDEX "routine_translation_routineId_language_key" ON "routine_translation"("routineId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_input_translation_routineInputId_language_key" ON "routine_input_translation"("routineInputId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_output_translation_routineOutputId_language_key" ON "routine_output_translation"("routineOutputId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_tags_taggedId_tagId_key" ON "routine_tags"("taggedId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "standard_createdByUserId_createdByOrganizationId_name_versi_key" ON "standard"("createdByUserId", "createdByOrganizationId", "name", "version");

-- CreateIndex
CREATE UNIQUE INDEX "standard_translation_standardId_language_key" ON "standard_translation"("standardId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "standard_tags_taggedId_tagId_key" ON "standard_tags"("taggedId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "tag_tag_key" ON "tag"("tag");

-- CreateIndex
CREATE UNIQUE INDEX "tag_translation_tagId_language_key" ON "tag_translation"("tagId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_confirmationCode_key" ON "user"("confirmationCode");

-- CreateIndex
CREATE UNIQUE INDEX "user_resetPasswordCode_key" ON "user"("resetPasswordCode");

-- CreateIndex
CREATE UNIQUE INDEX "user_handle_key" ON "user"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "user_translation_userId_language_key" ON "user_translation"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_language_userId_language_key" ON "user_language"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_roles_userId_roleId_key" ON "user_roles"("userId", "roleId");

-- CreateIndex
CREATE UNIQUE INDEX "user_tag_hidden_userId_tagId_key" ON "user_tag_hidden"("userId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "view_byId_organizationId_projectId_routineId_standardId_use_key" ON "view"("byId", "organizationId", "projectId", "routineId", "standardId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_stakingAddress_key" ON "wallet"("stakingAddress");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_publicAddress_key" ON "wallet"("publicAddress");

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment_translation" ADD CONSTRAINT "comment_translation_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "email" ADD CONSTRAINT "email_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "handle" ADD CONSTRAINT "handle_walletId_fkey" FOREIGN KEY ("walletId") REFERENCES "wallet"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node" ADD CONSTRAINT "node_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_translation" ADD CONSTRAINT "node_translation_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_end" ADD CONSTRAINT "node_end_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_fromId_fkey" FOREIGN KEY ("fromId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_toId_fkey" FOREIGN KEY ("toId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link_when" ADD CONSTRAINT "node_link_when_linkId_fkey" FOREIGN KEY ("linkId") REFERENCES "node_link"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link_when_translation" ADD CONSTRAINT "node_link_when_translation_whenId_fkey" FOREIGN KEY ("whenId") REFERENCES "node_link_when"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_loop" ADD CONSTRAINT "node_loop_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_loop_while" ADD CONSTRAINT "node_loop_while_loopId_fkey" FOREIGN KEY ("loopId") REFERENCES "node_loop"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_loop_while_translation" ADD CONSTRAINT "node_loop_while_translation_whileId_fkey" FOREIGN KEY ("whileId") REFERENCES "node_loop_while"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_routine_list" ADD CONSTRAINT "node_routine_list_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_routine_list_item" ADD CONSTRAINT "node_routine_list_item_listId_fkey" FOREIGN KEY ("listId") REFERENCES "node_routine_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_routine_list_item" ADD CONSTRAINT "node_routine_list_item_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_routine_list_item_translation" ADD CONSTRAINT "node_routine_list_item_translation_itemId_fkey" FOREIGN KEY ("itemId") REFERENCES "node_routine_list_item"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_translation" ADD CONSTRAINT "organization_translation_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_tags" ADD CONSTRAINT "organization_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_tags" ADD CONSTRAINT "organization_tags_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_users" ADD CONSTRAINT "organization_users_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_users" ADD CONSTRAINT "organization_users_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "project"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_translation" ADD CONSTRAINT "project_translation_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_tags" ADD CONSTRAINT "project_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_tags" ADD CONSTRAINT "project_tags_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_fromId_fkey" FOREIGN KEY ("fromId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource" ADD CONSTRAINT "resource_listId_fkey" FOREIGN KEY ("listId") REFERENCES "resource_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_translation" ADD CONSTRAINT "resource_translation_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list" ADD CONSTRAINT "resource_list_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list" ADD CONSTRAINT "resource_list_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list" ADD CONSTRAINT "resource_list_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list" ADD CONSTRAINT "resource_list_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list_translation" ADD CONSTRAINT "resource_list_translation_listId_fkey" FOREIGN KEY ("listId") REFERENCES "resource_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "routine"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_translation" ADD CONSTRAINT "routine_translation_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_input" ADD CONSTRAINT "routine_input_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_input" ADD CONSTRAINT "routine_input_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_input_translation" ADD CONSTRAINT "routine_input_translation_routineInputId_fkey" FOREIGN KEY ("routineInputId") REFERENCES "routine_input"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_output" ADD CONSTRAINT "routine_output_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_output" ADD CONSTRAINT "routine_output_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_output_translation" ADD CONSTRAINT "routine_output_translation_routineOutputId_fkey" FOREIGN KEY ("routineOutputId") REFERENCES "routine_output"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_tags" ADD CONSTRAINT "routine_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_tags" ADD CONSTRAINT "routine_tags_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_step" ADD CONSTRAINT "run_step_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_step" ADD CONSTRAINT "run_step_runId_fkey" FOREIGN KEY ("runId") REFERENCES "run"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdByOrganizationId_fkey" FOREIGN KEY ("createdByOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_translation" ADD CONSTRAINT "standard_translation_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_tags" ADD CONSTRAINT "standard_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_tags" ADD CONSTRAINT "standard_tags_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "star" ADD CONSTRAINT "star_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag" ADD CONSTRAINT "tag_createdByUserId_fkey" FOREIGN KEY ("createdByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag_translation" ADD CONSTRAINT "tag_translation_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_translation" ADD CONSTRAINT "user_translation_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_language" ADD CONSTRAINT "user_language_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_roles" ADD CONSTRAINT "user_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_roles" ADD CONSTRAINT "user_roles_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_tag_hidden" ADD CONSTRAINT "user_tag_hidden_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_tag_hidden" ADD CONSTRAINT "user_tag_hidden_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
