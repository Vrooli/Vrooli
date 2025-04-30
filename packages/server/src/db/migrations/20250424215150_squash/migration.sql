-- CreateExtension
CREATE EXTENSION IF NOT EXISTS "citext";

-- CreateExtension
CREATE EXTENSION IF NOT EXISTS "vector";

-- CreateEnum
CREATE TYPE "AccountStatus" AS ENUM ('Deleted', 'Unlocked', 'SoftLocked', 'HardLocked');

-- CreateEnum
CREATE TYPE "AwardCategory" AS ENUM ('AccountAnniversary', 'AccountNew', 'ApiCreate', 'CommentCreate', 'IssueCreate', 'NoteCreate', 'ObjectBookmark', 'ObjectReact', 'OrganizationCreate', 'OrganizationJoin', 'ProjectCreate', 'PullRequestCreate', 'PullRequestComplete', 'ReportEnd', 'ReportContribute', 'Reputation', 'RunRoutine', 'RunProject', 'RoutineCreate', 'SmartContractCreate', 'StandardCreate', 'Streak', 'UserInvite');

-- CreateEnum
CREATE TYPE "ChatInviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "CodeType" AS ENUM ('DataConvert', 'SmartContract');

-- CreateEnum
CREATE TYPE "IssueStatus" AS ENUM ('Draft', 'Open', 'Canceled', 'ClosedResolved', 'ClosedUnresolved', 'Rejected');

-- CreateEnum
CREATE TYPE "MemberInviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "MeetingInviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "NodeType" AS ENUM ('End', 'Redirect', 'RoutineList', 'Start');

-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('Pending', 'Paid', 'Failed');

-- CreateEnum
CREATE TYPE "PaymentType" AS ENUM ('Credits', 'Donation', 'PremiumMonthly', 'PremiumYearly');

-- CreateEnum
CREATE TYPE "PeriodType" AS ENUM ('Hourly', 'Daily', 'Weekly', 'Monthly', 'Yearly');

-- CreateEnum
CREATE TYPE "PullRequestStatus" AS ENUM ('Draft', 'Open', 'Canceled', 'Merged', 'Rejected');

-- CreateEnum
CREATE TYPE "ReportStatus" AS ENUM ('ClosedDeleted', 'ClosedFalseReport', 'ClosedHidden', 'ClosedNonIssue', 'ClosedSuspended', 'Open');

-- CreateEnum
CREATE TYPE "ReportSuggestedAction" AS ENUM ('Delete', 'FalseReport', 'HideUntilFixed', 'NonIssue', 'SuspendUser');

-- CreateEnum
CREATE TYPE "ResourceUsedFor" AS ENUM ('Community', 'Context', 'Developer', 'Donation', 'ExternalService', 'Feed', 'Install', 'Learning', 'Notes', 'OfficialWebsite', 'Proposal', 'Related', 'Researching', 'Scheduling', 'Social', 'Tutorial');

-- CreateEnum
CREATE TYPE "RoutineType" AS ENUM ('Action', 'Api', 'Code', 'Data', 'Generate', 'Informational', 'MultiStep', 'SmartContract');

-- CreateEnum
CREATE TYPE "RunStatus" AS ENUM ('Scheduled', 'InProgress', 'Paused', 'Completed', 'Failed', 'Cancelled');

-- CreateEnum
CREATE TYPE "RunStepStatus" AS ENUM ('InProgress', 'Completed', 'Skipped');

-- CreateEnum
CREATE TYPE "ScheduleRecurrenceType" AS ENUM ('Daily', 'Weekly', 'Monthly', 'Yearly');

-- CreateEnum
CREATE TYPE "StandardType" AS ENUM ('DataStructure', 'Prompt');

-- CreateEnum
CREATE TYPE "TransferStatus" AS ENUM ('Accepted', 'Denied', 'Pending');

-- CreateTable
CREATE TABLE "award" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "timeCurrentTierCompleted" TIMESTAMPTZ(6),
    "category" "AwardCategory" NOT NULL,
    "progress" INTEGER NOT NULL DEFAULT 0,
    "userId" UUID NOT NULL,

    CONSTRAINT "award_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,
    "parentId" UUID,

    CONSTRAINT "api_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "api_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "api_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "callLink" VARCHAR(1024) NOT NULL,
    "documentationLink" VARCHAR(1024),
    "schemaLanguage" VARCHAR(128),
    "schemaText" VARCHAR(16384),
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "rootId" UUID NOT NULL,
    "resourceListId" UUID,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),
    "pullRequestId" UUID,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "api_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128) NOT NULL,
    "summary" VARCHAR(1024),
    "details" VARCHAR(8192),
    "language" VARCHAR(3) NOT NULL,
    "apiVersionId" UUID NOT NULL,

    CONSTRAINT "api_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_key" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "creditsUsed" BIGINT NOT NULL DEFAULT 0,
    "disabledAt" TIMESTAMPTZ(6),
    "key" VARCHAR(255) NOT NULL,
    "limitHard" BIGINT NOT NULL DEFAULT 25000000000,
    "limitSoft" BIGINT DEFAULT 25000000000,
    "name" VARCHAR(128) NOT NULL,
    "permissions" VARCHAR(4096) NOT NULL,
    "stopAtLimit" BOOLEAN NOT NULL DEFAULT true,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "api_key_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_key_external" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "key" VARCHAR(255) NOT NULL,
    "disabledAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "service" VARCHAR(128) NOT NULL,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "api_key_external_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "bookmark" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "listId" UUID,
    "apiId" UUID,
    "codeId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "tagId" UUID,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "bookmark_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "bookmark_list" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "index" INTEGER NOT NULL,
    "label" VARCHAR(128) NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "bookmark_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "inviteId" UUID,
    "isPrivate" BOOLEAN NOT NULL DEFAULT true,
    "openToAnyoneWithInvite" BOOLEAN NOT NULL DEFAULT false,
    "creatorId" UUID,
    "teamId" UUID,

    CONSTRAINT "chat_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "chatId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "name" VARCHAR(128),
    "description" VARCHAR(2048),

    CONSTRAINT "chat_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_message" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "metadata" VARCHAR(4096),
    "score" INTEGER NOT NULL DEFAULT 0,
    "sequence" SERIAL NOT NULL,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "parentId" UUID,
    "userId" UUID,
    "chatId" UUID,

    CONSTRAINT "chat_message_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_message_translation" (
    "id" UUID NOT NULL,
    "text" VARCHAR(32768) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "messageId" UUID NOT NULL,

    CONSTRAINT "chat_message_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_participants" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasUnread" BOOLEAN NOT NULL DEFAULT true,
    "chatId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "chat_participants_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_invite" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "ChatInviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "chatId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "chat_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "chat_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_roles" (
    "id" UUID NOT NULL,
    "chatId" UUID NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "chat_roles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "parentId" UUID,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "code_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "codeLanguage" VARCHAR(128) NOT NULL,
    "codeType" "CodeType" NOT NULL DEFAULT 'DataConvert',
    "completedAt" TIMESTAMPTZ(6),
    "data" VARCHAR(8192),
    "default" VARCHAR(2048),
    "content" VARCHAR(8192) NOT NULL,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "resourceListId" UUID,
    "rootId" UUID NOT NULL,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "pullRequestId" UUID,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "code_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "jsonVariable" VARCHAR(8192),
    "codeVersionId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "code_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code_version_contract" (
    "id" UUID NOT NULL,
    "address" VARCHAR(256),
    "blockchain" VARCHAR(128) NOT NULL,
    "codeVersionId" UUID NOT NULL,
    "contractType" VARCHAR(256),
    "hash" VARCHAR(256),
    "isAddressVerified" BOOLEAN NOT NULL DEFAULT false,
    "isHashVerified" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "code_version_contract_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "code_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "code_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "code_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,
    "apiVersionId" UUID,
    "codeVersionId" UUID,
    "issueId" UUID,
    "noteVersionId" UUID,
    "parentId" UUID,
    "projectVersionId" UUID,
    "pullRequestId" UUID,
    "routineVersionId" UUID,
    "standardVersionId" UUID,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "comment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment_translation" (
    "id" UUID NOT NULL,
    "text" VARCHAR(32768) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "commentId" UUID NOT NULL,

    CONSTRAINT "comment_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "email" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "emailAddress" CITEXT NOT NULL,
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(256),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "email_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "issue" (
    "id" UUID NOT NULL,
    "status" "IssueStatus" NOT NULL DEFAULT 'Open',
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "closedAt" TIMESTAMPTZ(6),
    "hasBeenClosedOrRejected" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "apiId" UUID,
    "codeId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "teamId" UUID,
    "closedById" UUID,
    "createdById" UUID,
    "referencedVersionId" UUID,

    CONSTRAINT "issue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "issue_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "issue_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "issue_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "issueId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "issue_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "label" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "label" VARCHAR(128) NOT NULL,
    "color" VARCHAR(7),
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "label_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "label_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "labelId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "label_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "parentId" UUID,
    "createdById" UUID,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,
    "createdByTeamId" UUID,

    CONSTRAINT "note_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "note_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "note_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT true,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "rootId" UUID NOT NULL,
    "pullRequestId" UUID,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),

    CONSTRAINT "note_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "language" VARCHAR(3) NOT NULL,
    "noteVersionId" UUID NOT NULL,

    CONSTRAINT "note_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note_page" (
    "id" UUID NOT NULL,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "pageIndex" INTEGER NOT NULL DEFAULT 0,
    "text" VARCHAR(65536) NOT NULL,
    "translationId" UUID NOT NULL,

    CONSTRAINT "note_page_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notification" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "category" VARCHAR(64) NOT NULL,
    "isRead" BOOLEAN NOT NULL DEFAULT false,
    "title" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "count" INTEGER NOT NULL DEFAULT 1,
    "link" VARCHAR(2048),
    "imgLink" VARCHAR(2048),
    "userId" UUID NOT NULL,

    CONSTRAINT "notification_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "push_device" (
    "id" UUID NOT NULL,
    "endpoint" VARCHAR(1024) NOT NULL,
    "p256dh" VARCHAR(1024) NOT NULL,
    "auth" VARCHAR(1024) NOT NULL,
    "expires" TIMESTAMPTZ(6),
    "name" VARCHAR(128),
    "userId" UUID NOT NULL,

    CONSTRAINT "push_device_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notification_subscription" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "context" VARCHAR(2048),
    "silent" BOOLEAN NOT NULL DEFAULT false,
    "apiId" UUID,
    "chatId" UUID,
    "codeId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "meetingId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "pullRequestId" UUID,
    "reportId" UUID,
    "routineId" UUID,
    "scheduleId" UUID,
    "standardId" UUID,
    "teamId" UUID,
    "subscriberId" UUID NOT NULL,

    CONSTRAINT "notification_subscription_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "bannerImage" VARCHAR(2048),
    "handle" CITEXT,
    "isOpenToNewMembers" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "profileImage" VARCHAR(2048),
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "parentId" UUID,
    "premiumId" UUID,
    "resourceListId" UUID,
    "createdById" UUID,
    "stripeCustomerId" VARCHAR(255),

    CONSTRAINT "team_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team_language" (
    "id" UUID NOT NULL,
    "teamId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "team_language_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "inviteId" UUID,
    "scheduleId" UUID,
    "openToAnyoneWithInvite" BOOLEAN NOT NULL DEFAULT false,
    "showOnTeamProfile" BOOLEAN NOT NULL DEFAULT false,
    "teamId" UUID NOT NULL,

    CONSTRAINT "meeting_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_attendees" (
    "id" UUID NOT NULL,
    "meetingId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "meeting_attendees_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_invite" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "MeetingInviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "meetingId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "meeting_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "meeting_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_roles" (
    "id" UUID NOT NULL,
    "meetingId" UUID NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "meeting_roles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "meetingId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "name" VARCHAR(128),
    "description" VARCHAR(2048),
    "link" VARCHAR(2048),

    CONSTRAINT "meeting_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "bio" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "teamId" UUID NOT NULL,

    CONSTRAINT "team_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "team_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isAdmin" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "teamId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "member_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member_invite" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "MemberInviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "willBeAdmin" BOOLEAN NOT NULL DEFAULT false,
    "willHavePermissions" VARCHAR(4096),
    "teamId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "member_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "phone" (
    "id" UUID NOT NULL,
    "phoneNumber" VARCHAR(16) NOT NULL,
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(16),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "phone_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "payment" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "amount" INTEGER NOT NULL,
    "checkoutId" VARCHAR(255) NOT NULL,
    "currency" VARCHAR(255) NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "paymentMethod" VARCHAR(255) NOT NULL,
    "paymentType" "PaymentType" NOT NULL DEFAULT 'PremiumMonthly',
    "status" "PaymentStatus" NOT NULL DEFAULT 'Pending',
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "payment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "premium" (
    "id" UUID NOT NULL,
    "credits" BIGINT NOT NULL DEFAULT 0,
    "customPlan" VARCHAR(2048),
    "enabledAt" TIMESTAMPTZ(6),
    "expiresAt" TIMESTAMPTZ(6),
    "hasReceivedFreeTrial" BOOLEAN NOT NULL DEFAULT false,
    "isActive" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "premium_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "handle" CITEXT,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,
    "parentId" UUID,

    CONSTRAINT "project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 1,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "isComplete" BOOLEAN NOT NULL DEFAULT true,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "rootId" UUID NOT NULL,
    "simplicity" INTEGER NOT NULL DEFAULT 1,
    "timesStarted" INTEGER NOT NULL DEFAULT 0,
    "timesCompleted" INTEGER NOT NULL DEFAULT 0,
    "resourceListId" UUID,
    "pullRequestId" UUID,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),

    CONSTRAINT "project_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version_directory" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isRoot" BOOLEAN NOT NULL DEFAULT false,
    "parentDirectoryId" UUID,
    "childOrder" VARCHAR(4096),
    "projectVersionId" UUID NOT NULL,

    CONSTRAINT "project_version_directory_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version_directory_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "projectVersionDirectoryId" UUID NOT NULL,

    CONSTRAINT "project_version_directory_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "projectVersionId" UUID NOT NULL,

    CONSTRAINT "project_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version_end_next" (
    "id" UUID NOT NULL,
    "fromProjectVersionId" UUID NOT NULL,
    "toProjectVersionId" UUID NOT NULL,

    CONSTRAINT "project_version_end_next_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "project_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "project_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "pull_request" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "PullRequestStatus" NOT NULL DEFAULT 'Open',
    "hasBeenClosedOrRejected" BOOLEAN NOT NULL DEFAULT false,
    "mergedOrRejectedAt" TIMESTAMPTZ(6),
    "createdById" UUID,
    "toApiId" UUID,
    "toCodeId" UUID,
    "toNoteId" UUID,
    "toProjectId" UUID,
    "toRoutineId" UUID,
    "toStandardId" UUID,

    CONSTRAINT "pull_request_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "pull_request_translation" (
    "id" UUID NOT NULL,
    "text" VARCHAR(32768) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "pullRequestId" UUID NOT NULL,

    CONSTRAINT "pull_request_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reaction" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "emoji" VARCHAR(32) NOT NULL,
    "byId" UUID NOT NULL,
    "apiId" UUID,
    "chatMessageId" UUID,
    "codeId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,

    CONSTRAINT "reaction_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reaction_summary" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "emoji" VARCHAR(32) NOT NULL,
    "count" INTEGER NOT NULL DEFAULT 0,
    "apiId" UUID,
    "chatMessageId" UUID,
    "codeId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,

    CONSTRAINT "reaction_summary_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_list" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "userId" UUID NOT NULL,

    CONSTRAINT "reminder_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "dueDate" TIMESTAMPTZ(6),
    "index" INTEGER NOT NULL,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "reminderListId" UUID NOT NULL,

    CONSTRAINT "reminder_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_item" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "dueDate" TIMESTAMPTZ(6),
    "index" INTEGER NOT NULL,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "reminderId" UUID NOT NULL,

    CONSTRAINT "reminder_item_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "reason" VARCHAR(128) NOT NULL,
    "details" VARCHAR(8192),
    "language" VARCHAR(3) NOT NULL,
    "status" "ReportStatus" NOT NULL,
    "apiVersionId" UUID,
    "chatMessageId" UUID,
    "codeVersionId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteVersionId" UUID,
    "projectVersionId" UUID,
    "routineVersionId" UUID,
    "standardVersionId" UUID,
    "tagId" UUID,
    "teamId" UUID,
    "userId" UUID,
    "createdById" UUID,

    CONSTRAINT "report_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report_response" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "reportId" UUID NOT NULL,
    "createdById" UUID NOT NULL,
    "actionSuggested" "ReportSuggestedAction" NOT NULL,
    "details" VARCHAR(8192),
    "language" VARCHAR(3),

    CONSTRAINT "report_response_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reputation_history" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "amount" INTEGER NOT NULL,
    "event" VARCHAR(128) NOT NULL,
    "objectId1" UUID,
    "objectId2" UUID,
    "userId" UUID NOT NULL,

    CONSTRAINT "reputation_history_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "index" INTEGER DEFAULT 0,
    "link" VARCHAR(1024) NOT NULL,
    "usedFor" "ResourceUsedFor" NOT NULL DEFAULT 'Context',
    "listId" UUID NOT NULL,

    CONSTRAINT "resource_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "resourceId" UUID NOT NULL,

    CONSTRAINT "resource_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_list" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "resource_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_list_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "listId" UUID NOT NULL,

    CONSTRAINT "resource_list_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "role" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "permissions" VARCHAR(4096) NOT NULL,
    "teamId" UUID NOT NULL,

    CONSTRAINT "role_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "role_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "role_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isInternal" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "parentId" UUID,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "config" VARCHAR(32768),
    "complexity" INTEGER NOT NULL DEFAULT 1,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "isAutomatable" BOOLEAN NOT NULL DEFAULT false,
    "isComplete" BOOLEAN NOT NULL DEFAULT true,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "rootId" UUID NOT NULL,
    "routineType" "RoutineType" NOT NULL DEFAULT 'Informational',
    "simplicity" INTEGER NOT NULL DEFAULT 1,
    "timesStarted" INTEGER NOT NULL DEFAULT 0,
    "timesCompleted" INTEGER NOT NULL DEFAULT 0,
    "resourceListId" UUID,
    "apiVersionId" UUID,
    "codeVersionId" UUID,
    "pullRequestId" UUID,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),

    CONSTRAINT "routine_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_subroutine" (
    "id" UUID NOT NULL,
    "parentRoutineId" UUID NOT NULL,
    "subroutineId" UUID NOT NULL,

    CONSTRAINT "routine_version_subroutine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "description" VARCHAR(2048),
    "instructions" VARCHAR(8192),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "routineVersionId" UUID NOT NULL,

    CONSTRAINT "routine_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_input" (
    "id" UUID NOT NULL,
    "index" INTEGER DEFAULT 0,
    "isRequired" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128),
    "routineVersionId" UUID NOT NULL,
    "standardVersionId" UUID,

    CONSTRAINT "routine_version_input_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_input_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "helpText" VARCHAR(2048),
    "routineVersionInputId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "routine_version_input_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_output" (
    "id" UUID NOT NULL,
    "index" INTEGER DEFAULT 0,
    "name" VARCHAR(128),
    "routineVersionId" UUID NOT NULL,
    "standardVersionId" UUID,

    CONSTRAINT "routine_version_output_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_verstion_output_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "helpText" VARCHAR(2048),
    "routineOutputId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "routine_verstion_output_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_end_next" (
    "id" UUID NOT NULL,
    "fromRoutineVersionId" UUID NOT NULL,
    "toRoutineVersionId" UUID NOT NULL,

    CONSTRAINT "routine_version_end_next_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "routine_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "routine_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "data" VARCHAR(16384),
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "scheduleId" UUID,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT 'Scheduled',
    "projectVersionId" UUID,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "run_project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project_step" (
    "id" UUID NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(128) NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,
    "directoryInId" UUID NOT NULL,
    "startedAt" TIMESTAMPTZ(6),
    "status" "RunStepStatus" NOT NULL DEFAULT 'InProgress',
    "timeElapsed" INTEGER,
    "directoryId" UUID,
    "runProjectId" UUID NOT NULL,

    CONSTRAINT "run_project_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "data" VARCHAR(16384),
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "scheduleId" UUID,
    "wasRunAutomatically" BOOLEAN NOT NULL DEFAULT false,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT 'Scheduled',
    "routineVersionId" UUID,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "run_routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_io" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "data" VARCHAR(8192) NOT NULL,
    "nodeInputName" VARCHAR(128) NOT NULL,
    "nodeName" VARCHAR(128) NOT NULL,
    "runRoutineId" UUID NOT NULL,
    "routineVersionInputId" UUID,
    "routineVersionOutputId" UUID,

    CONSTRAINT "run_routine_io_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_step" (
    "id" UUID NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(128) NOT NULL,
    "nodeId" UUID NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,
    "subroutineInId" UUID NOT NULL,
    "startedAt" TIMESTAMPTZ(6),
    "status" "RunStepStatus" NOT NULL DEFAULT 'InProgress',
    "timeElapsed" INTEGER,
    "runRoutineId" UUID NOT NULL,
    "subroutineId" UUID,

    CONSTRAINT "run_routine_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "startTime" TIMESTAMP(3) NOT NULL,
    "endTime" TIMESTAMP(3) NOT NULL,
    "timezone" TEXT NOT NULL,

    CONSTRAINT "schedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "schedule_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule_exception" (
    "id" UUID NOT NULL,
    "scheduleId" UUID NOT NULL,
    "originalStartTime" TIMESTAMP(3) NOT NULL,
    "newStartTime" TIMESTAMP(3),
    "newEndTime" TIMESTAMP(3),

    CONSTRAINT "schedule_exception_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule_recurrence" (
    "id" UUID NOT NULL,
    "scheduleId" UUID NOT NULL,
    "recurrenceType" "ScheduleRecurrenceType" NOT NULL,
    "interval" INTEGER NOT NULL DEFAULT 1,
    "dayOfWeek" INTEGER,
    "dayOfMonth" INTEGER,
    "month" INTEGER,
    "endDate" TIMESTAMP(3),
    "duration" INTEGER,

    CONSTRAINT "schedule_recurrence_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasBeenTransferred" BOOLEAN NOT NULL DEFAULT false,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isInternal" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "parentId" UUID,
    "createdById" UUID,
    "ownedByTeamId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "standard_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_version" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "codeLanguage" VARCHAR(128) NOT NULL DEFAULT 'json',
    "default" VARCHAR(2048),
    "variant" "StandardType" NOT NULL DEFAULT 'DataStructure',
    "props" VARCHAR(8192) NOT NULL,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "resourceListId" UUID,
    "rootId" UUID NOT NULL,
    "yup" VARCHAR(8192),
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "pullRequestId" UUID,
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "isFile" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "standard_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_version_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "jsonVariable" VARCHAR(8192),
    "standardVersionId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "standard_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "standard_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "standard_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_site" (
    "id" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "activeUsers" INTEGER NOT NULL,
    "apiCalls" INTEGER NOT NULL,
    "apisCreated" INTEGER NOT NULL,
    "codesCreated" INTEGER NOT NULL,
    "codesCompleted" INTEGER NOT NULL,
    "codeCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "codeCalls" INTEGER NOT NULL,
    "projectsCreated" INTEGER NOT NULL,
    "projectsCompleted" INTEGER NOT NULL,
    "projectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "routinesCreated" INTEGER NOT NULL,
    "routinesCompleted" INTEGER NOT NULL,
    "routineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "routineSimplicityAverage" DOUBLE PRECISION NOT NULL,
    "routineComplexityAverage" DOUBLE PRECISION NOT NULL,
    "runProjectsStarted" INTEGER NOT NULL,
    "runProjectsCompleted" INTEGER NOT NULL,
    "runProjectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runProjectContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "runRoutinesStarted" INTEGER NOT NULL,
    "runRoutinesCompleted" INTEGER NOT NULL,
    "runRoutineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runRoutineContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "standardsCreated" INTEGER NOT NULL,
    "standardsCompleted" INTEGER NOT NULL,
    "standardCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "teamsCreated" INTEGER NOT NULL,
    "verifiedEmailsCreated" INTEGER NOT NULL,
    "verifiedWalletsCreated" INTEGER NOT NULL,

    CONSTRAINT "stats_site_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_api" (
    "id" UUID NOT NULL,
    "apiId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "calls" INTEGER NOT NULL,
    "routineVersions" INTEGER NOT NULL,

    CONSTRAINT "stats_api_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_code" (
    "id" UUID NOT NULL,
    "codeId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "calls" INTEGER NOT NULL,
    "routineVersions" INTEGER NOT NULL,

    CONSTRAINT "stats_code_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_project" (
    "id" UUID NOT NULL,
    "projectId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "directories" INTEGER NOT NULL,
    "apis" INTEGER NOT NULL,
    "codes" INTEGER NOT NULL,
    "notes" INTEGER NOT NULL,
    "projects" INTEGER NOT NULL,
    "routines" INTEGER NOT NULL,
    "standards" INTEGER NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "teams" INTEGER NOT NULL,

    CONSTRAINT "stats_project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_routine" (
    "id" UUID NOT NULL,
    "routineId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_standard" (
    "id" UUID NOT NULL,
    "standardId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "linksToInputs" INTEGER NOT NULL,
    "linksToOutputs" INTEGER NOT NULL,

    CONSTRAINT "stats_standard_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_team" (
    "id" UUID NOT NULL,
    "teamId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "apis" INTEGER NOT NULL,
    "codes" INTEGER NOT NULL,
    "members" INTEGER NOT NULL,
    "notes" INTEGER NOT NULL,
    "projects" INTEGER NOT NULL,
    "routines" INTEGER NOT NULL,
    "runRoutinesStarted" INTEGER NOT NULL,
    "runRoutinesCompleted" INTEGER NOT NULL,
    "runRoutineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runRoutineContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "standards" INTEGER NOT NULL,

    CONSTRAINT "stats_team_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_user" (
    "id" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "apisCreated" INTEGER NOT NULL,
    "codesCreated" INTEGER NOT NULL,
    "codesCompleted" INTEGER NOT NULL,
    "codeCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "projectsCreated" INTEGER NOT NULL,
    "projectsCompleted" INTEGER NOT NULL,
    "projectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "routinesCreated" INTEGER NOT NULL,
    "routinesCompleted" INTEGER NOT NULL,
    "routineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runProjectsStarted" INTEGER NOT NULL,
    "runProjectsCompleted" INTEGER NOT NULL,
    "runProjectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runProjectContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "runRoutinesStarted" INTEGER NOT NULL,
    "runRoutinesCompleted" INTEGER NOT NULL,
    "runRoutineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runRoutineContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "standardsCreated" INTEGER NOT NULL,
    "standardsCompleted" INTEGER NOT NULL,
    "standardCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "teamsCreated" INTEGER NOT NULL,

    CONSTRAINT "stats_user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tag" VARCHAR(128) NOT NULL,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "createdById" UUID,

    CONSTRAINT "tag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
    "description" VARCHAR(2048),
    "tagId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "tag_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "transfer" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "TransferStatus" NOT NULL DEFAULT 'Pending',
    "initializedByReceiver" BOOLEAN NOT NULL DEFAULT false,
    "message" VARCHAR(4096),
    "denyReason" VARCHAR(2048),
    "fromTeamId" UUID,
    "fromUserId" UUID,
    "toTeamId" UUID,
    "toUserId" UUID,
    "apiId" UUID,
    "codeId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,

    CONSTRAINT "transfer_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "bannerImage" VARCHAR(2048),
    "confirmationCode" VARCHAR(256),
    "confirmationCodeDate" TIMESTAMPTZ(6),
    "invitedByUserId" UUID,
    "isBot" BOOLEAN NOT NULL DEFAULT false,
    "isBotDepictingPerson" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateApis" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateApisCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateCodes" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateCodesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateNotes" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateNotesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateMemberships" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateProjects" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateProjectsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivatePullRequests" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoles" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoutines" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoutinesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateStandards" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateStandardsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateTeamsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateBookmarks" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateVotes" BOOLEAN NOT NULL DEFAULT false,
    "lastExport" TIMESTAMPTZ(6),
    "lastLoginAttempt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "logInAttempts" INTEGER NOT NULL DEFAULT 0,
    "numExports" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(128) NOT NULL,
    "profileImage" VARCHAR(2048),
    "theme" VARCHAR(255) NOT NULL DEFAULT 'light',
    "handle" CITEXT,
    "currentStreak" INTEGER NOT NULL DEFAULT 0,
    "longestStreak" INTEGER NOT NULL DEFAULT 0,
    "accountTabsOrder" VARCHAR(255),
    "botSettings" VARCHAR(4096),
    "notificationSettings" VARCHAR(2048),
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "reputation" INTEGER NOT NULL DEFAULT 0,
    "premiumId" UUID,
    "stripeCustomerId" VARCHAR(255),
    "status" "AccountStatus" NOT NULL DEFAULT 'Unlocked',

    CONSTRAINT "user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_auth" (
    "id" UUID NOT NULL,
    "user_id" UUID NOT NULL,
    "provider" TEXT NOT NULL,
    "provider_user_id" TEXT,
    "hashed_password" TEXT,
    "resetPasswordCode" VARCHAR(256),
    "lastResetPasswordRequestAttempt" TIMESTAMPTZ(6),
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,

    CONSTRAINT "user_auth_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "session" (
    "id" UUID NOT NULL,
    "user_id" UUID NOT NULL,
    "auth_id" UUID NOT NULL,
    "last_refresh_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "expires_at" TIMESTAMPTZ(6) NOT NULL,
    "revoked" BOOLEAN NOT NULL DEFAULT false,
    "device_info" VARCHAR(1024),
    "ip_address" VARCHAR(45),

    CONSTRAINT "session_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_translation" (
    "id" UUID NOT NULL,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "embedding" vector(768),
    "embeddingNeedsUpdate" BOOLEAN NOT NULL DEFAULT true,
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
CREATE TABLE "view" (
    "id" UUID NOT NULL,
    "lastViewedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "byId" UUID NOT NULL,
    "apiId" UUID,
    "codeId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "standardId" UUID,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "view_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "wallet" (
    "id" UUID NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "stakingAddress" VARCHAR(128) NOT NULL,
    "publicAddress" VARCHAR(128),
    "name" VARCHAR(128),
    "nonce" VARCHAR(8092),
    "nonceCreationTime" TIMESTAMPTZ(6),
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "wasReported" BOOLEAN NOT NULL DEFAULT false,
    "teamId" UUID,
    "userId" UUID,

    CONSTRAINT "wallet_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "_api_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_api_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_code_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_code_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_note_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_note_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_memberTorole" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_memberTorole_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_project_version_directory_listing" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_project_version_directory_listing_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_project_version_directoryToroutine_version" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_project_version_directoryToroutine_version_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_project_version_directoryTostandard_version" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_project_version_directoryTostandard_version_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateTable
CREATE TABLE "_project_version_directoryToteam" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL,

    CONSTRAINT "_project_version_directoryToteam_AB_pkey" PRIMARY KEY ("A","B")
);

-- CreateIndex
CREATE UNIQUE INDEX "award_userId_category_key" ON "award"("userId", "category");

-- CreateIndex
CREATE UNIQUE INDEX "api_labels_labelledId_labelId_key" ON "api_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "api_tags_taggedId_tagTag_key" ON "api_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_callLink_key" ON "api_version"("callLink");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_resourceListId_key" ON "api_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_pullRequestId_key" ON "api_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_rootId_versionIndex_key" ON "api_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "api_key_key_key" ON "api_key"("key");

-- CreateIndex
CREATE UNIQUE INDEX "bookmark_list_label_key" ON "bookmark_list"("label");

-- CreateIndex
CREATE UNIQUE INDEX "chat_translation_chatId_language_key" ON "chat_translation"("chatId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "chat_message_translation_messageId_language_key" ON "chat_message_translation"("messageId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "chat_participants_chatId_userId_key" ON "chat_participants"("chatId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_invite_chatId_userId_key" ON "chat_invite"("chatId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_labels_labelledId_labelId_key" ON "chat_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_roles_chatId_roleId_key" ON "chat_roles"("chatId", "roleId");

-- CreateIndex
CREATE UNIQUE INDEX "code_version_resourceListId_key" ON "code_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "code_version_pullRequestId_key" ON "code_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "code_version_rootId_versionIndex_key" ON "code_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "code_version_translation_codeVersionId_language_key" ON "code_version_translation"("codeVersionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "code_version_contract_codeVersionId_key" ON "code_version_contract"("codeVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "code_tags_taggedId_tagTag_key" ON "code_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "code_labels_labelledId_labelId_key" ON "code_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "comment_translation_commentId_language_key" ON "comment_translation"("commentId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "email_emailAddress_key" ON "email"("emailAddress");

-- CreateIndex
CREATE UNIQUE INDEX "issue_labels_labelledId_labelId_key" ON "issue_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "issue_translation_issueId_language_key" ON "issue_translation"("issueId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "label_label_key" ON "label"("label");

-- CreateIndex
CREATE UNIQUE INDEX "label_label_ownedByUserId_ownedByTeamId_key" ON "label"("label", "ownedByUserId", "ownedByTeamId");

-- CreateIndex
CREATE UNIQUE INDEX "label_translation_labelId_language_key" ON "label_translation"("labelId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "note_labels_labelledId_labelId_key" ON "note_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "note_tags_taggedId_tagTag_key" ON "note_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "note_version_pullRequestId_key" ON "note_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "note_version_rootId_versionIndex_key" ON "note_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "note_page_translationId_pageIndex_key" ON "note_page"("translationId", "pageIndex");

-- CreateIndex
CREATE UNIQUE INDEX "push_device_endpoint_key" ON "push_device"("endpoint");

-- CreateIndex
CREATE UNIQUE INDEX "team_handle_key" ON "team"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "team_premiumId_key" ON "team"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "team_resourceListId_key" ON "team"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "team_stripeCustomerId_key" ON "team"("stripeCustomerId");

-- CreateIndex
CREATE UNIQUE INDEX "team_language_teamId_language_key" ON "team_language"("teamId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_attendees_meetingId_userId_key" ON "meeting_attendees"("meetingId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_invite_meetingId_userId_key" ON "meeting_invite"("meetingId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_labels_labelledId_labelId_key" ON "meeting_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_roles_meetingId_roleId_key" ON "meeting_roles"("meetingId", "roleId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_translation_meetingId_language_key" ON "meeting_translation"("meetingId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "team_translation_teamId_language_key" ON "team_translation"("teamId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "team_tags_taggedId_tagTag_key" ON "team_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "member_teamId_userId_key" ON "member"("teamId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "member_invite_userId_key" ON "member_invite"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "member_invite_userId_teamId_key" ON "member_invite"("userId", "teamId");

-- CreateIndex
CREATE UNIQUE INDEX "phone_phoneNumber_key" ON "phone"("phoneNumber");

-- CreateIndex
CREATE UNIQUE INDEX "project_handle_key" ON "project"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_resourceListId_key" ON "project_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_pullRequestId_key" ON "project_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_rootId_versionIndex_key" ON "project_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_directory_translation_projectVersionDirecto_key" ON "project_version_directory_translation"("projectVersionDirectoryId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_translation_projectVersionId_language_key" ON "project_version_translation"("projectVersionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "project_version_end_next_fromProjectVersionId_toProjectVers_key" ON "project_version_end_next"("fromProjectVersionId", "toProjectVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "project_tags_taggedId_tagTag_key" ON "project_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "project_labels_labelledId_labelId_key" ON "project_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "pull_request_translation_pullRequestId_language_key" ON "pull_request_translation"("pullRequestId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "reaction_summary_emoji_apiId_chatMessageId_codeId_commentId_key" ON "reaction_summary"("emoji", "apiId", "chatMessageId", "codeId", "commentId", "issueId", "noteId", "projectId", "routineId", "standardId");

-- CreateIndex
CREATE UNIQUE INDEX "report_response_reportId_createdById_key" ON "report_response"("reportId", "createdById");

-- CreateIndex
CREATE UNIQUE INDEX "resource_translation_resourceId_language_key" ON "resource_translation"("resourceId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "resource_list_translation_listId_language_key" ON "resource_list_translation"("listId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "role_teamId_name_key" ON "role"("teamId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "role_translation_roleId_language_key" ON "role_translation"("roleId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_resourceListId_key" ON "routine_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_pullRequestId_key" ON "routine_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_rootId_versionIndex_key" ON "routine_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_subroutine_parentRoutineId_subroutineId_key" ON "routine_version_subroutine"("parentRoutineId", "subroutineId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_translation_routineVersionId_language_key" ON "routine_version_translation"("routineVersionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_input_translation_routineVersionInputId_lan_key" ON "routine_version_input_translation"("routineVersionInputId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_verstion_output_translation_routineOutputId_languag_key" ON "routine_verstion_output_translation"("routineOutputId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_end_next_fromRoutineVersionId_toRoutineVers_key" ON "routine_version_end_next"("fromRoutineVersionId", "toRoutineVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_tags_taggedId_tagTag_key" ON "routine_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "routine_labels_labelledId_labelId_key" ON "routine_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "schedule_labels_labelledId_labelId_key" ON "schedule_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "standard_version_resourceListId_key" ON "standard_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "standard_version_pullRequestId_key" ON "standard_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "standard_version_rootId_versionIndex_key" ON "standard_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "standard_version_translation_standardVersionId_language_key" ON "standard_version_translation"("standardVersionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "standard_tags_taggedId_tagTag_key" ON "standard_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "standard_labels_labelledId_labelId_key" ON "standard_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "tag_tag_key" ON "tag"("tag");

-- CreateIndex
CREATE UNIQUE INDEX "tag_translation_tagId_language_key" ON "tag_translation"("tagId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_confirmationCode_key" ON "user"("confirmationCode");

-- CreateIndex
CREATE UNIQUE INDEX "user_handle_key" ON "user"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "user_premiumId_key" ON "user"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "user_stripeCustomerId_key" ON "user"("stripeCustomerId");

-- CreateIndex
CREATE UNIQUE INDEX "user_auth_resetPasswordCode_key" ON "user_auth"("resetPasswordCode");

-- CreateIndex
CREATE INDEX "user_auth_user_id_idx" ON "user_auth"("user_id");

-- CreateIndex
CREATE UNIQUE INDEX "user_auth_provider_provider_user_id_key" ON "user_auth"("provider", "provider_user_id");

-- CreateIndex
CREATE INDEX "session_user_id_idx" ON "session"("user_id");

-- CreateIndex
CREATE INDEX "session_auth_id_idx" ON "session"("auth_id");

-- CreateIndex
CREATE UNIQUE INDEX "user_translation_userId_language_key" ON "user_translation"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_language_userId_language_key" ON "user_language"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_stakingAddress_key" ON "wallet"("stakingAddress");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_publicAddress_key" ON "wallet"("publicAddress");

-- CreateIndex
CREATE INDEX "_api_versionToproject_version_directory_B_index" ON "_api_versionToproject_version_directory"("B");

-- CreateIndex
CREATE INDEX "_code_versionToproject_version_directory_B_index" ON "_code_versionToproject_version_directory"("B");

-- CreateIndex
CREATE INDEX "_note_versionToproject_version_directory_B_index" ON "_note_versionToproject_version_directory"("B");

-- CreateIndex
CREATE INDEX "_memberTorole_B_index" ON "_memberTorole"("B");

-- CreateIndex
CREATE INDEX "_project_version_directory_listing_B_index" ON "_project_version_directory_listing"("B");

-- CreateIndex
CREATE INDEX "_project_version_directoryToroutine_version_B_index" ON "_project_version_directoryToroutine_version"("B");

-- CreateIndex
CREATE INDEX "_project_version_directoryTostandard_version_B_index" ON "_project_version_directoryTostandard_version"("B");

-- CreateIndex
CREATE INDEX "_project_version_directoryToteam_B_index" ON "_project_version_directoryToteam"("B");

-- AddForeignKey
ALTER TABLE "award" ADD CONSTRAINT "award_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "api_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_labels" ADD CONSTRAINT "api_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_labels" ADD CONSTRAINT "api_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_tags" ADD CONSTRAINT "api_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_tags" ADD CONSTRAINT "api_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version_translation" ADD CONSTRAINT "api_version_translation_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key" ADD CONSTRAINT "api_key_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key" ADD CONSTRAINT "api_key_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key_external" ADD CONSTRAINT "api_key_external_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key_external" ADD CONSTRAINT "api_key_external_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_listId_fkey" FOREIGN KEY ("listId") REFERENCES "bookmark_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark_list" ADD CONSTRAINT "bookmark_list_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat" ADD CONSTRAINT "chat_creatorId_fkey" FOREIGN KEY ("creatorId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat" ADD CONSTRAINT "chat_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_translation" ADD CONSTRAINT "chat_translation_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_message" ADD CONSTRAINT "chat_message_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_message" ADD CONSTRAINT "chat_message_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "chat_message"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_message" ADD CONSTRAINT "chat_message_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_message_translation" ADD CONSTRAINT "chat_message_translation_messageId_fkey" FOREIGN KEY ("messageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_participants" ADD CONSTRAINT "chat_participants_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_participants" ADD CONSTRAINT "chat_participants_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_invite" ADD CONSTRAINT "chat_invite_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_invite" ADD CONSTRAINT "chat_invite_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_labels" ADD CONSTRAINT "chat_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_labels" ADD CONSTRAINT "chat_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_roles" ADD CONSTRAINT "chat_roles_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_roles" ADD CONSTRAINT "chat_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code" ADD CONSTRAINT "code_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code" ADD CONSTRAINT "code_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "code_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code" ADD CONSTRAINT "code_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code" ADD CONSTRAINT "code_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_version" ADD CONSTRAINT "code_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_version" ADD CONSTRAINT "code_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_version" ADD CONSTRAINT "code_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_version_translation" ADD CONSTRAINT "code_version_translation_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_version_contract" ADD CONSTRAINT "code_version_contract_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_tags" ADD CONSTRAINT "code_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_tags" ADD CONSTRAINT "code_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_labels" ADD CONSTRAINT "code_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "code_labels" ADD CONSTRAINT "code_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_noteVersionId_fkey" FOREIGN KEY ("noteVersionId") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment_translation" ADD CONSTRAINT "comment_translation_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "email" ADD CONSTRAINT "email_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "email" ADD CONSTRAINT "email_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_closedById_fkey" FOREIGN KEY ("closedById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_labels" ADD CONSTRAINT "issue_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_labels" ADD CONSTRAINT "issue_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_translation" ADD CONSTRAINT "issue_translation_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label" ADD CONSTRAINT "label_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label" ADD CONSTRAINT "label_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label_translation" ADD CONSTRAINT "label_translation_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "note_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_labels" ADD CONSTRAINT "note_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_labels" ADD CONSTRAINT "note_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_tags" ADD CONSTRAINT "note_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_tags" ADD CONSTRAINT "note_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_version" ADD CONSTRAINT "note_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_version" ADD CONSTRAINT "note_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_version_translation" ADD CONSTRAINT "note_version_translation_noteVersionId_fkey" FOREIGN KEY ("noteVersionId") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note_page" ADD CONSTRAINT "note_page_translationId_fkey" FOREIGN KEY ("translationId") REFERENCES "note_version_translation"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification" ADD CONSTRAINT "notification_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "push_device" ADD CONSTRAINT "push_device_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_reportId_fkey" FOREIGN KEY ("reportId") REFERENCES "report"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_subscriberId_fkey" FOREIGN KEY ("subscriberId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_premiumId_fkey" FOREIGN KEY ("premiumId") REFERENCES "premium"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_language" ADD CONSTRAINT "team_language_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting" ADD CONSTRAINT "meeting_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting" ADD CONSTRAINT "meeting_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_attendees" ADD CONSTRAINT "meeting_attendees_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_attendees" ADD CONSTRAINT "meeting_attendees_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_invite" ADD CONSTRAINT "meeting_invite_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_invite" ADD CONSTRAINT "meeting_invite_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_labels" ADD CONSTRAINT "meeting_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_labels" ADD CONSTRAINT "meeting_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_roles" ADD CONSTRAINT "meeting_roles_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_roles" ADD CONSTRAINT "meeting_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_translation" ADD CONSTRAINT "meeting_translation_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_translation" ADD CONSTRAINT "team_translation_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_tags" ADD CONSTRAINT "team_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_tags" ADD CONSTRAINT "team_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member" ADD CONSTRAINT "member_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member" ADD CONSTRAINT "member_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member_invite" ADD CONSTRAINT "member_invite_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member_invite" ADD CONSTRAINT "member_invite_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "phone" ADD CONSTRAINT "phone_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "phone" ADD CONSTRAINT "phone_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "payment_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "payment_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "project_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version" ADD CONSTRAINT "project_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version" ADD CONSTRAINT "project_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version" ADD CONSTRAINT "project_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_directory" ADD CONSTRAINT "project_version_directory_parentDirectoryId_fkey" FOREIGN KEY ("parentDirectoryId") REFERENCES "project_version_directory"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_directory" ADD CONSTRAINT "project_version_directory_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_directory_translation" ADD CONSTRAINT "project_version_directory_translation_projectVersionDirect_fkey" FOREIGN KEY ("projectVersionDirectoryId") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_translation" ADD CONSTRAINT "project_version_translation_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_end_next" ADD CONSTRAINT "project_version_end_next_fromProjectVersionId_fkey" FOREIGN KEY ("fromProjectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_version_end_next" ADD CONSTRAINT "project_version_end_next_toProjectVersionId_fkey" FOREIGN KEY ("toProjectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_tags" ADD CONSTRAINT "project_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_tags" ADD CONSTRAINT "project_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_labels" ADD CONSTRAINT "project_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project_labels" ADD CONSTRAINT "project_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toApiId_fkey" FOREIGN KEY ("toApiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toCodeId_fkey" FOREIGN KEY ("toCodeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toNoteId_fkey" FOREIGN KEY ("toNoteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toProjectId_fkey" FOREIGN KEY ("toProjectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toRoutineId_fkey" FOREIGN KEY ("toRoutineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toStandardId_fkey" FOREIGN KEY ("toStandardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request_translation" ADD CONSTRAINT "pull_request_translation_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder_list" ADD CONSTRAINT "reminder_list_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder" ADD CONSTRAINT "reminder_reminderListId_fkey" FOREIGN KEY ("reminderListId") REFERENCES "reminder_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder_item" ADD CONSTRAINT "reminder_item_reminderId_fkey" FOREIGN KEY ("reminderId") REFERENCES "reminder"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_noteVersionId_fkey" FOREIGN KEY ("noteVersionId") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report_response" ADD CONSTRAINT "report_response_reportId_fkey" FOREIGN KEY ("reportId") REFERENCES "report"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report_response" ADD CONSTRAINT "report_response_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reputation_history" ADD CONSTRAINT "reputation_history_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource" ADD CONSTRAINT "resource_listId_fkey" FOREIGN KEY ("listId") REFERENCES "resource_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_translation" ADD CONSTRAINT "resource_translation_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_list_translation" ADD CONSTRAINT "resource_list_translation_listId_fkey" FOREIGN KEY ("listId") REFERENCES "resource_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "role" ADD CONSTRAINT "role_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "role_translation" ADD CONSTRAINT "role_translation_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_codeVersionId_fkey" FOREIGN KEY ("codeVersionId") REFERENCES "code_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_subroutine" ADD CONSTRAINT "routine_version_subroutine_parentRoutineId_fkey" FOREIGN KEY ("parentRoutineId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_subroutine" ADD CONSTRAINT "routine_version_subroutine_subroutineId_fkey" FOREIGN KEY ("subroutineId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_translation" ADD CONSTRAINT "routine_version_translation_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_input" ADD CONSTRAINT "routine_version_input_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_input" ADD CONSTRAINT "routine_version_input_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_input_translation" ADD CONSTRAINT "routine_version_input_translation_routineVersionInputId_fkey" FOREIGN KEY ("routineVersionInputId") REFERENCES "routine_version_input"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_output" ADD CONSTRAINT "routine_version_output_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_output" ADD CONSTRAINT "routine_version_output_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_verstion_output_translation" ADD CONSTRAINT "routine_verstion_output_translation_routineOutputId_fkey" FOREIGN KEY ("routineOutputId") REFERENCES "routine_version_output"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_end_next" ADD CONSTRAINT "routine_version_end_next_fromRoutineVersionId_fkey" FOREIGN KEY ("fromRoutineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version_end_next" ADD CONSTRAINT "routine_version_end_next_toRoutineVersionId_fkey" FOREIGN KEY ("toRoutineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_tags" ADD CONSTRAINT "routine_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_tags" ADD CONSTRAINT "routine_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_labels" ADD CONSTRAINT "routine_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_labels" ADD CONSTRAINT "routine_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_step" ADD CONSTRAINT "run_project_step_directoryId_fkey" FOREIGN KEY ("directoryId") REFERENCES "project_version_directory"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_step" ADD CONSTRAINT "run_project_step_runProjectId_fkey" FOREIGN KEY ("runProjectId") REFERENCES "run_project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_routineVersionInputId_fkey" FOREIGN KEY ("routineVersionInputId") REFERENCES "routine_version_input"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_io" ADD CONSTRAINT "run_routine_io_routineVersionOutputId_fkey" FOREIGN KEY ("routineVersionOutputId") REFERENCES "routine_version_output"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_subroutineId_fkey" FOREIGN KEY ("subroutineId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_labels" ADD CONSTRAINT "schedule_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_labels" ADD CONSTRAINT "schedule_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_exception" ADD CONSTRAINT "schedule_exception_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_recurrence" ADD CONSTRAINT "schedule_recurrence_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "standard_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_version" ADD CONSTRAINT "standard_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_version" ADD CONSTRAINT "standard_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_version" ADD CONSTRAINT "standard_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_version_translation" ADD CONSTRAINT "standard_version_translation_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_tags" ADD CONSTRAINT "standard_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_tags" ADD CONSTRAINT "standard_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_labels" ADD CONSTRAINT "standard_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard_labels" ADD CONSTRAINT "standard_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_api" ADD CONSTRAINT "stats_api_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_code" ADD CONSTRAINT "stats_code_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_project" ADD CONSTRAINT "stats_project_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_routine" ADD CONSTRAINT "stats_routine_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_standard" ADD CONSTRAINT "stats_standard_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_team" ADD CONSTRAINT "stats_team_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_user" ADD CONSTRAINT "stats_user_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag" ADD CONSTRAINT "tag_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag_translation" ADD CONSTRAINT "tag_translation_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_fromTeamId_fkey" FOREIGN KEY ("fromTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_fromUserId_fkey" FOREIGN KEY ("fromUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_toTeamId_fkey" FOREIGN KEY ("toTeamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_toUserId_fkey" FOREIGN KEY ("toUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user" ADD CONSTRAINT "user_invitedByUserId_fkey" FOREIGN KEY ("invitedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user" ADD CONSTRAINT "user_premiumId_fkey" FOREIGN KEY ("premiumId") REFERENCES "premium"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_auth" ADD CONSTRAINT "user_auth_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "session" ADD CONSTRAINT "session_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "session" ADD CONSTRAINT "session_auth_id_fkey" FOREIGN KEY ("auth_id") REFERENCES "user_auth"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_translation" ADD CONSTRAINT "user_translation_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_language" ADD CONSTRAINT "user_language_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_codeId_fkey" FOREIGN KEY ("codeId") REFERENCES "code"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_api_versionToproject_version_directory" ADD CONSTRAINT "_api_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_api_versionToproject_version_directory" ADD CONSTRAINT "_api_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_code_versionToproject_version_directory" ADD CONSTRAINT "_code_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_code_versionToproject_version_directory" ADD CONSTRAINT "_code_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_note_versionToproject_version_directory" ADD CONSTRAINT "_note_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_note_versionToproject_version_directory" ADD CONSTRAINT "_note_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_memberTorole" ADD CONSTRAINT "_memberTorole_A_fkey" FOREIGN KEY ("A") REFERENCES "member"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_memberTorole" ADD CONSTRAINT "_memberTorole_B_fkey" FOREIGN KEY ("B") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directory_listing" ADD CONSTRAINT "_project_version_directory_listing_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directory_listing" ADD CONSTRAINT "_project_version_directory_listing_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryToroutine_version" ADD CONSTRAINT "_project_version_directoryToroutine_version_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryToroutine_version" ADD CONSTRAINT "_project_version_directoryToroutine_version_B_fkey" FOREIGN KEY ("B") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryTostandard_version" ADD CONSTRAINT "_project_version_directoryTostandard_version_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryTostandard_version" ADD CONSTRAINT "_project_version_directoryTostandard_version_B_fkey" FOREIGN KEY ("B") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryToteam" ADD CONSTRAINT "_project_version_directoryToteam_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryToteam" ADD CONSTRAINT "_project_version_directoryToteam_B_fkey" FOREIGN KEY ("B") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;
