-- CreateEnum
CREATE TYPE "AccountStatus" AS ENUM ('Deleted', 'Unlocked', 'SoftLocked', 'HardLocked');

-- CreateEnum
CREATE TYPE "InviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "IssueStatus" AS ENUM ('Draft', 'Open', 'Canceled', 'ClosedResolved', 'ClosedUnresolved', 'Rejected');

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
CREATE TYPE "RunStatus" AS ENUM ('Scheduled', 'InProgress', 'Paused', 'Completed', 'Failed', 'Cancelled');

-- CreateEnum
CREATE TYPE "RunStepStatus" AS ENUM ('InProgress', 'Completed', 'Skipped');

-- CreateEnum
CREATE TYPE "ScheduleRecurrenceType" AS ENUM ('Daily', 'Weekly', 'Monthly', 'Yearly');

-- CreateEnum
CREATE TYPE "TransferStatus" AS ENUM ('Accepted', 'Denied', 'Pending');

-- CreateTable
CREATE TABLE "award" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "tierCompletedAt" TIMESTAMPTZ(6),
    "category" VARCHAR(128) NOT NULL,
    "progress" INTEGER NOT NULL DEFAULT 0,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "award_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_key" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "creditsUsed" BIGINT NOT NULL DEFAULT 0,
    "disabledAt" TIMESTAMPTZ(6),
    "key" VARCHAR(255) NOT NULL,
    "limitHard" BIGINT NOT NULL DEFAULT 25000000000,
    "limitSoft" BIGINT DEFAULT 25000000000,
    "name" VARCHAR(128) NOT NULL,
    "permissions" JSONB DEFAULT '{}',
    "stopAtLimit" BOOLEAN NOT NULL DEFAULT true,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "api_key_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_key_external" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "key" VARCHAR(255) NOT NULL,
    "disabledAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "service" VARCHAR(128) NOT NULL,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "api_key_external_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "bookmark" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "listId" BIGINT,
    "resourceId" BIGINT,
    "commentId" BIGINT,
    "issueId" BIGINT,
    "tagId" BIGINT,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "bookmark_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "bookmark_list" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "label" VARCHAR(128) NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "bookmark_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "isPrivate" BOOLEAN NOT NULL DEFAULT true,
    "openToAnyoneWithInvite" BOOLEAN NOT NULL DEFAULT false,
    "creatorId" BIGINT,
    "teamId" BIGINT,

    CONSTRAINT "chat_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "chatId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "name" VARCHAR(128),
    "description" VARCHAR(2048),

    CONSTRAINT "chat_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_message" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "metadata" VARCHAR(4096),
    "text" VARCHAR(32768) NOT NULL,
    "score" INTEGER NOT NULL DEFAULT 0,
    "sequence" SERIAL NOT NULL,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "parentId" BIGINT,
    "userId" BIGINT,
    "chatId" BIGINT,

    CONSTRAINT "chat_message_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_participants" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "hasUnread" BOOLEAN NOT NULL DEFAULT true,
    "chatId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "chat_participants_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chat_invite" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "status" "InviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "chatId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "chat_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "ownedByTeamId" BIGINT,
    "ownedByUserId" BIGINT,
    "resourceVersionId" BIGINT,
    "issueId" BIGINT,
    "parentId" BIGINT,
    "pullRequestId" BIGINT,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "comment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment_translation" (
    "id" BIGINT NOT NULL,
    "text" VARCHAR(32768) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "commentId" BIGINT NOT NULL,

    CONSTRAINT "comment_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "email" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "emailAddress" CITEXT NOT NULL,
    "verifiedAt" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(256),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "email_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "issue" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "status" "IssueStatus" NOT NULL DEFAULT 'Open',
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "closedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "resourceId" BIGINT,
    "teamId" BIGINT,
    "closedById" BIGINT,
    "createdById" BIGINT,

    CONSTRAINT "issue_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "issue_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "issueId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "issue_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "inviteId" BIGINT,
    "scheduleId" BIGINT,
    "openToAnyoneWithInvite" BOOLEAN NOT NULL DEFAULT false,
    "showOnTeamProfile" BOOLEAN NOT NULL DEFAULT false,
    "teamId" BIGINT NOT NULL,

    CONSTRAINT "meeting_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_attendees" (
    "id" BIGINT NOT NULL,
    "meetingId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "meeting_attendees_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_invite" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "status" "InviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "meetingId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "meeting_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "meetingId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "name" VARCHAR(128),
    "description" VARCHAR(2048),
    "link" VARCHAR(2048),

    CONSTRAINT "meeting_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notification" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "category" VARCHAR(64) NOT NULL,
    "isRead" BOOLEAN NOT NULL DEFAULT false,
    "title" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "count" INTEGER NOT NULL DEFAULT 1,
    "link" VARCHAR(2048),
    "imgLink" VARCHAR(2048),
    "userId" BIGINT NOT NULL,

    CONSTRAINT "notification_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notification_subscription" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "context" VARCHAR(2048),
    "silent" BOOLEAN NOT NULL DEFAULT false,
    "resourceId" BIGINT,
    "chatId" BIGINT,
    "commentId" BIGINT,
    "issueId" BIGINT,
    "meetingId" BIGINT,
    "pullRequestId" BIGINT,
    "reportId" BIGINT,
    "scheduleId" BIGINT,
    "subscriberId" BIGINT NOT NULL,
    "teamId" BIGINT,

    CONSTRAINT "notification_subscription_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "push_device" (
    "id" BIGINT NOT NULL,
    "endpoint" VARCHAR(1024) NOT NULL,
    "p256dh" VARCHAR(1024) NOT NULL,
    "auth" VARCHAR(1024) NOT NULL,
    "expires" TIMESTAMPTZ(6),
    "name" VARCHAR(128),
    "userId" BIGINT NOT NULL,

    CONSTRAINT "push_device_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "bannerImage" VARCHAR(2048),
    "config" JSONB DEFAULT '{}',
    "handle" CITEXT,
    "isOpenToNewMembers" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "languages" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "permissions" JSONB DEFAULT '{}',
    "profileImage" VARCHAR(2048),
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "parentId" BIGINT,
    "premiumId" BIGINT,
    "createdById" BIGINT,
    "stripeCustomerId" VARCHAR(255),

    CONSTRAINT "team_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "bio" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "teamId" BIGINT NOT NULL,

    CONSTRAINT "team_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "team_tag" (
    "id" BIGINT NOT NULL,
    "taggedId" BIGINT NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "team_tag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "isAdmin" BOOLEAN NOT NULL DEFAULT false,
    "permissions" JSONB DEFAULT '{}',
    "teamId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "member_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member_invite" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "status" "InviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "willBeAdmin" BOOLEAN NOT NULL DEFAULT false,
    "willHavePermissions" JSONB DEFAULT '{}',
    "teamId" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "member_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "phone" (
    "id" BIGINT NOT NULL,
    "phoneNumber" VARCHAR(16) NOT NULL,
    "verifiedAt" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(16),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "phone_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "payment" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "amount" INTEGER NOT NULL,
    "checkoutId" VARCHAR(255) NOT NULL,
    "currency" VARCHAR(255) NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "paymentMethod" VARCHAR(255) NOT NULL,
    "paymentType" "PaymentType" NOT NULL DEFAULT 'PremiumMonthly',
    "status" "PaymentStatus" NOT NULL DEFAULT 'Pending',
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "payment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "premium" (
    "id" BIGINT NOT NULL,
    "credits" BIGINT NOT NULL DEFAULT 0,
    "customPlan" VARCHAR(2048),
    "enabledAt" TIMESTAMPTZ(6),
    "expiresAt" TIMESTAMPTZ(6),
    "receivedFreeTrialAt" TIMESTAMPTZ(6),

    CONSTRAINT "premium_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "pull_request" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "status" "PullRequestStatus" NOT NULL DEFAULT 'Open',
    "closedAt" TIMESTAMPTZ(6),
    "createdById" BIGINT,
    "toResourceId" BIGINT,

    CONSTRAINT "pull_request_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "pull_request_translation" (
    "id" BIGINT NOT NULL,
    "text" VARCHAR(32768) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "pullRequestId" BIGINT NOT NULL,

    CONSTRAINT "pull_request_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reaction" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "emoji" VARCHAR(32) NOT NULL,
    "byId" BIGINT NOT NULL,
    "resourceId" BIGINT,
    "chatMessageId" BIGINT,
    "commentId" BIGINT,
    "issueId" BIGINT,

    CONSTRAINT "reaction_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reaction_summary" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "emoji" VARCHAR(32) NOT NULL,
    "count" INTEGER NOT NULL DEFAULT 0,
    "resourceId" BIGINT,
    "chatMessageId" BIGINT,
    "commentId" BIGINT,
    "issueId" BIGINT,

    CONSTRAINT "reaction_summary_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_list" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "reminder_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "dueDate" TIMESTAMPTZ(6),
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "index" INTEGER NOT NULL,
    "reminderListId" BIGINT NOT NULL,

    CONSTRAINT "reminder_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_item" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "dueDate" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "index" INTEGER NOT NULL,
    "reminderId" BIGINT NOT NULL,

    CONSTRAINT "reminder_item_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "reason" VARCHAR(128) NOT NULL,
    "details" VARCHAR(8192),
    "language" VARCHAR(3) NOT NULL,
    "status" "ReportStatus" NOT NULL,
    "resourceVersionId" BIGINT,
    "chatMessageId" BIGINT,
    "commentId" BIGINT,
    "issueId" BIGINT,
    "tagId" BIGINT,
    "teamId" BIGINT,
    "userId" BIGINT,
    "createdById" BIGINT,

    CONSTRAINT "report_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report_response" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "reportId" BIGINT NOT NULL,
    "createdById" BIGINT NOT NULL,
    "actionSuggested" "ReportSuggestedAction" NOT NULL,
    "details" VARCHAR(8192),
    "language" VARCHAR(3),

    CONSTRAINT "report_response_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reputation_history" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "amount" INTEGER NOT NULL,
    "event" VARCHAR(128) NOT NULL,
    "objectId1" BIGINT,
    "objectId2" BIGINT,
    "userId" BIGINT NOT NULL,

    CONSTRAINT "reputation_history_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "resourceType" VARCHAR(32) NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "transferredAt" TIMESTAMPTZ(6),
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "permissions" JSONB DEFAULT '{}',
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "createdById" BIGINT,
    "ownedByTeamId" BIGINT,
    "ownedByUserId" BIGINT,
    "parentId" BIGINT,
    "isInternal" BOOLEAN,

    CONSTRAINT "resource_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_version" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "rootId" BIGINT NOT NULL,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "config" JSONB DEFAULT '{}',
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isLatestPublic" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "versionNotes" VARCHAR(4096),
    "pullRequestId" BIGINT,
    "codeLanguage" VARCHAR(128),
    "resourceSubType" VARCHAR(32),
    "isAutomatable" BOOLEAN,
    "complexity" INTEGER,
    "simplicity" INTEGER,
    "timesStarted" INTEGER NOT NULL DEFAULT 0,
    "timesCompleted" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "resource_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_version_relation" (
    "id" BIGINT NOT NULL,
    "fromVersionId" BIGINT NOT NULL,
    "toVersionId" BIGINT NOT NULL,
    "relationType" VARCHAR(64) NOT NULL,
    "labels" TEXT[],

    CONSTRAINT "resource_version_relation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_translation" (
    "id" BIGINT NOT NULL,
    "resourceVersionId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "details" VARCHAR(8192),
    "instructions" VARCHAR(8192),

    CONSTRAINT "resource_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "resource_tag" (
    "id" BIGINT NOT NULL,
    "taggedId" BIGINT NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "resource_tag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "data" VARCHAR(16384),
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "scheduleId" BIGINT,
    "wasRunAutomatically" BOOLEAN NOT NULL DEFAULT false,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT 'Scheduled',
    "resourceVersionId" BIGINT,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "run_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_io" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "data" VARCHAR(8192) NOT NULL,
    "nodeInputName" VARCHAR(128) NOT NULL,
    "nodeName" VARCHAR(128) NOT NULL,
    "runId" BIGINT NOT NULL,

    CONSTRAINT "run_io_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_step" (
    "id" BIGINT NOT NULL,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(128) NOT NULL,
    "nodeId" UUID NOT NULL,
    "order" INTEGER NOT NULL DEFAULT 0,
    "resourceInId" UUID NOT NULL,
    "startedAt" TIMESTAMPTZ(6),
    "status" "RunStepStatus" NOT NULL DEFAULT 'InProgress',
    "timeElapsed" INTEGER,
    "runId" BIGINT NOT NULL,
    "resourceVersionId" BIGINT,

    CONSTRAINT "run_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "startTime" TIMESTAMP(3) NOT NULL,
    "endTime" TIMESTAMP(3) NOT NULL,
    "timezone" TEXT NOT NULL,

    CONSTRAINT "schedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule_exception" (
    "id" BIGINT NOT NULL,
    "scheduleId" BIGINT NOT NULL,
    "originalStartTime" TIMESTAMP(3) NOT NULL,
    "newStartTime" TIMESTAMP(3),
    "newEndTime" TIMESTAMP(3),

    CONSTRAINT "schedule_exception_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "schedule_recurrence" (
    "id" BIGINT NOT NULL,
    "scheduleId" BIGINT NOT NULL,
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
CREATE TABLE "stats_resource" (
    "id" BIGINT NOT NULL,
    "resourceId" BIGINT NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "references" INTEGER NOT NULL,
    "referencedBy" INTEGER NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_resource_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_site" (
    "id" BIGINT NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "activeUsers" INTEGER NOT NULL,
    "teamsCreated" INTEGER NOT NULL,
    "verifiedEmailsCreated" INTEGER NOT NULL,
    "verifiedWalletsCreated" INTEGER NOT NULL,
    "resourcesCreatedByType" JSONB,
    "resourcesCompletedByType" JSONB,
    "resourceCompletionTimeAverageByType" JSONB,
    "routineSimplicityAverage" DOUBLE PRECISION NOT NULL,
    "routineComplexityAverage" DOUBLE PRECISION NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_site_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_team" (
    "id" BIGINT NOT NULL,
    "teamId" BIGINT NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "resources" INTEGER NOT NULL,
    "members" INTEGER NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_team_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_user" (
    "id" BIGINT NOT NULL,
    "userId" BIGINT NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "resourcesCreatedByType" JSONB,
    "resourcesCompletedByType" JSONB,
    "resourceCompletionTimeAverageByType" JSONB,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "teamsCreated" INTEGER NOT NULL,

    CONSTRAINT "stats_user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "tag" VARCHAR(128) NOT NULL,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "createdById" BIGINT,

    CONSTRAINT "tag_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "description" VARCHAR(2048),
    "tagId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "tag_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "transfer" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "status" "TransferStatus" NOT NULL DEFAULT 'Pending',
    "initializedByReceiver" BOOLEAN NOT NULL DEFAULT false,
    "message" VARCHAR(4096),
    "denyReason" VARCHAR(2048),
    "fromTeamId" BIGINT,
    "fromUserId" BIGINT,
    "toTeamId" BIGINT,
    "toUserId" BIGINT,
    "resourceId" BIGINT,

    CONSTRAINT "transfer_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user" (
    "id" BIGINT NOT NULL,
    "publicId" VARCHAR(12) NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "bannerImage" VARCHAR(2048),
    "confirmationCode" VARCHAR(256),
    "confirmationCodeDate" TIMESTAMPTZ(6),
    "invitedByUserId" BIGINT,
    "isBot" BOOLEAN NOT NULL DEFAULT false,
    "isBotDepictingPerson" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateMemberships" BOOLEAN NOT NULL DEFAULT false,
    "isPrivatePullRequests" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateResources" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateResourcesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateTeamsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateBookmarks" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateVotes" BOOLEAN NOT NULL DEFAULT false,
    "languages" TEXT[] DEFAULT ARRAY[]::TEXT[],
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
    "premiumId" BIGINT,
    "stripeCustomerId" VARCHAR(255),
    "status" "AccountStatus" NOT NULL DEFAULT 'Unlocked',

    CONSTRAINT "user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_auth" (
    "id" BIGINT NOT NULL,
    "user_id" BIGINT NOT NULL,
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
    "id" BIGINT NOT NULL,
    "user_id" BIGINT NOT NULL,
    "auth_id" BIGINT NOT NULL,
    "last_refresh_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "expires_at" TIMESTAMPTZ(6) NOT NULL,
    "revokedAt" TIMESTAMPTZ(6),
    "device_info" VARCHAR(1024),
    "ip_address" VARCHAR(45),

    CONSTRAINT "session_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_translation" (
    "id" BIGINT NOT NULL,
    "embedding" vector(1536),
    "embeddingExpiredAt" TIMESTAMPTZ(6) DEFAULT CURRENT_TIMESTAMP,
    "bio" VARCHAR(2048),
    "userId" BIGINT NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "user_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "view" (
    "id" BIGINT NOT NULL,
    "lastViewedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "byId" BIGINT NOT NULL,
    "issueId" BIGINT,
    "resourceId" BIGINT,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "view_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "wallet" (
    "id" BIGINT NOT NULL,
    "createdAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMPTZ(6) NOT NULL,
    "stakingAddress" VARCHAR(128) NOT NULL,
    "publicAddress" VARCHAR(128),
    "name" VARCHAR(128),
    "nonce" VARCHAR(8092),
    "nonceCreationTime" TIMESTAMPTZ(6),
    "verifiedAt" TIMESTAMPTZ(6),
    "wasReported" BOOLEAN NOT NULL DEFAULT false,
    "teamId" BIGINT,
    "userId" BIGINT,

    CONSTRAINT "wallet_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "idx_award_userId" ON "award"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "award_userId_category_key" ON "award"("userId", "category");

-- CreateIndex
CREATE UNIQUE INDEX "api_key_key_key" ON "api_key"("key");

-- CreateIndex
CREATE INDEX "idx_api_key_userId" ON "api_key"("userId");

-- CreateIndex
CREATE INDEX "idx_api_key_teamId" ON "api_key"("teamId");

-- CreateIndex
CREATE INDEX "idx_api_key_external_userId" ON "api_key_external"("userId");

-- CreateIndex
CREATE INDEX "idx_api_key_external_teamId" ON "api_key_external"("teamId");

-- CreateIndex
CREATE UNIQUE INDEX "api_key_external_userId_service_name_key" ON "api_key_external"("userId", "service", "name");

-- CreateIndex
CREATE INDEX "idx_bookmark_listId_createdAt" ON "bookmark"("listId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_commentId_createdAt" ON "bookmark"("commentId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_issueId_createdAt" ON "bookmark"("issueId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_resourceId_createdAt" ON "bookmark"("resourceId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_tagId_createdAt" ON "bookmark"("tagId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_teamId_createdAt" ON "bookmark"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_userId_createdAt" ON "bookmark"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_bookmark_list_userId" ON "bookmark_list"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "bookmark_list_label_userId_key" ON "bookmark_list"("label", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_publicId_key" ON "chat"("publicId");

-- CreateIndex
CREATE INDEX "idx_chat_publicId" ON "chat"("publicId");

-- CreateIndex
CREATE INDEX "idx_chat_creatorId_createdAt" ON "chat"("creatorId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_chat_creatorId_updatedAt" ON "chat"("creatorId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_chat_teamId_createdAt" ON "chat"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_chat_teamId_updatedAt" ON "chat"("teamId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_chat_translation_chatId" ON "chat_translation"("chatId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_translation_chatId_language_key" ON "chat_translation"("chatId", "language");

-- CreateIndex
CREATE INDEX "idx_chat_message_chatId_createdAt" ON "chat_message"("chatId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_chat_participants_chatId" ON "chat_participants"("chatId");

-- CreateIndex
CREATE INDEX "idx_chat_participants_userId" ON "chat_participants"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_participants_chatId_userId_key" ON "chat_participants"("chatId", "userId");

-- CreateIndex
CREATE INDEX "idx_chat_invite_chatId" ON "chat_invite"("chatId");

-- CreateIndex
CREATE INDEX "idx_chat_invite_userId" ON "chat_invite"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "chat_invite_chatId_userId_key" ON "chat_invite"("chatId", "userId");

-- CreateIndex
CREATE INDEX "idx_comment_issueId_createdAt" ON "comment"("issueId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_comment_ownedByTeamId_createdAt" ON "comment"("ownedByTeamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_comment_ownedByUserId_createdAt" ON "comment"("ownedByUserId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_comment_parentId" ON "comment"("parentId");

-- CreateIndex
CREATE INDEX "idx_comment_pullRequestId_createdAt" ON "comment"("pullRequestId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_comment_resourceVersionId_createdAt" ON "comment"("resourceVersionId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_comment_score" ON "comment"("score");

-- CreateIndex
CREATE INDEX "idx_comment_translation_commentId" ON "comment_translation"("commentId");

-- CreateIndex
CREATE UNIQUE INDEX "comment_translation_commentId_language_key" ON "comment_translation"("commentId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "email_emailAddress_key" ON "email"("emailAddress");

-- CreateIndex
CREATE INDEX "idx_email_teamId" ON "email"("teamId");

-- CreateIndex
CREATE INDEX "idx_email_userId" ON "email"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "issue_publicId_key" ON "issue"("publicId");

-- CreateIndex
CREATE INDEX "idx_issue_publicId" ON "issue"("publicId");

-- CreateIndex
CREATE INDEX "idx_issue_teamId_createdAt" ON "issue"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_issue_teamId_updatedAt" ON "issue"("teamId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_issue_closedById_createdAt" ON "issue"("closedById", "createdAt");

-- CreateIndex
CREATE INDEX "idx_issue_closedById_updatedAt" ON "issue"("closedById", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_issue_createdById_createdAt" ON "issue"("createdById", "createdAt");

-- CreateIndex
CREATE INDEX "idx_issue_createdById_updatedAt" ON "issue"("createdById", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_issue_resourceId_createdAt" ON "issue"("resourceId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_issue_resourceId_updatedAt" ON "issue"("resourceId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_issue_translation_issueId" ON "issue_translation"("issueId");

-- CreateIndex
CREATE UNIQUE INDEX "issue_translation_issueId_language_key" ON "issue_translation"("issueId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_publicId_key" ON "meeting"("publicId");

-- CreateIndex
CREATE INDEX "idx_meeting_inviteId" ON "meeting"("inviteId");

-- CreateIndex
CREATE INDEX "idx_meeting_publicId" ON "meeting"("publicId");

-- CreateIndex
CREATE INDEX "idx_meeting_scheduleId" ON "meeting"("scheduleId");

-- CreateIndex
CREATE INDEX "idx_meeting_teamId_createdAt" ON "meeting"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_meeting_teamId_updatedAt" ON "meeting"("teamId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_meeting_attendees_meetingId" ON "meeting_attendees"("meetingId");

-- CreateIndex
CREATE INDEX "idx_meeting_attendees_userId" ON "meeting_attendees"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_attendees_meetingId_userId_key" ON "meeting_attendees"("meetingId", "userId");

-- CreateIndex
CREATE INDEX "idx_meeting_invite_meetingId" ON "meeting_invite"("meetingId");

-- CreateIndex
CREATE INDEX "idx_meeting_invite_userId" ON "meeting_invite"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_invite_meetingId_userId_key" ON "meeting_invite"("meetingId", "userId");

-- CreateIndex
CREATE INDEX "idx_meeting_translation_meetingId" ON "meeting_translation"("meetingId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_translation_meetingId_language_key" ON "meeting_translation"("meetingId", "language");

-- CreateIndex
CREATE INDEX "idx_notification_userId_createdAt" ON "notification"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_notification_userId_isRead" ON "notification"("userId", "isRead");

-- CreateIndex
CREATE INDEX "idx_notification_userId_category" ON "notification"("userId", "category");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_chatId" ON "notification_subscription"("chatId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_commentId" ON "notification_subscription"("commentId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_issueId" ON "notification_subscription"("issueId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_meetingId" ON "notification_subscription"("meetingId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_pullRequestId" ON "notification_subscription"("pullRequestId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_reportId" ON "notification_subscription"("reportId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_resourceId" ON "notification_subscription"("resourceId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_scheduleId" ON "notification_subscription"("scheduleId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_subscriberId" ON "notification_subscription"("subscriberId");

-- CreateIndex
CREATE INDEX "idx_notification_subscription_teamId" ON "notification_subscription"("teamId");

-- CreateIndex
CREATE UNIQUE INDEX "push_device_endpoint_key" ON "push_device"("endpoint");

-- CreateIndex
CREATE INDEX "idx_push_device_userId" ON "push_device"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "team_publicId_key" ON "team"("publicId");

-- CreateIndex
CREATE UNIQUE INDEX "team_handle_key" ON "team"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "team_premiumId_key" ON "team"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "team_stripeCustomerId_key" ON "team"("stripeCustomerId");

-- CreateIndex
CREATE INDEX "idx_team_createdById" ON "team"("createdById");

-- CreateIndex
CREATE INDEX "idx_team_parentId" ON "team"("parentId");

-- CreateIndex
CREATE INDEX "idx_team_publicId" ON "team"("publicId");

-- CreateIndex
CREATE INDEX "idx_team_translation_teamId" ON "team_translation"("teamId");

-- CreateIndex
CREATE UNIQUE INDEX "team_translation_teamId_language_key" ON "team_translation"("teamId", "language");

-- CreateIndex
CREATE INDEX "idx_team_tag_taggedId" ON "team_tag"("taggedId");

-- CreateIndex
CREATE UNIQUE INDEX "team_tag_taggedId_tagTag_key" ON "team_tag"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "member_publicId_key" ON "member"("publicId");

-- CreateIndex
CREATE INDEX "idx_member_publicId" ON "member"("publicId");

-- CreateIndex
CREATE INDEX "idx_member_teamId_createdAt" ON "member"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_member_teamId_updatedAt" ON "member"("teamId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_member_userId" ON "member"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "member_teamId_userId_key" ON "member"("teamId", "userId");

-- CreateIndex
CREATE INDEX "idx_member_invite_userId" ON "member_invite"("userId");

-- CreateIndex
CREATE INDEX "idx_member_invite_teamId" ON "member_invite"("teamId");

-- CreateIndex
CREATE UNIQUE INDEX "member_invite_userId_teamId_key" ON "member_invite"("userId", "teamId");

-- CreateIndex
CREATE UNIQUE INDEX "phone_phoneNumber_key" ON "phone"("phoneNumber");

-- CreateIndex
CREATE INDEX "idx_phone_userId" ON "phone"("userId");

-- CreateIndex
CREATE INDEX "idx_phone_teamId" ON "phone"("teamId");

-- CreateIndex
CREATE INDEX "idx_payment_teamId" ON "payment"("teamId");

-- CreateIndex
CREATE INDEX "idx_payment_userId" ON "payment"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "pull_request_publicId_key" ON "pull_request"("publicId");

-- CreateIndex
CREATE INDEX "idx_pull_request_publicId" ON "pull_request"("publicId");

-- CreateIndex
CREATE INDEX "idx_pull_request_createdById_createdAt" ON "pull_request"("createdById", "createdAt");

-- CreateIndex
CREATE INDEX "idx_pull_request_toResourceId_createdAt" ON "pull_request"("toResourceId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_pull_request_translation_pullRequestId" ON "pull_request_translation"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "pull_request_translation_pullRequestId_language_key" ON "pull_request_translation"("pullRequestId", "language");

-- CreateIndex
CREATE INDEX "idx_reaction_byId_createdAt" ON "reaction"("byId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_resourceId_createdAt" ON "reaction"("resourceId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_chatMessageId_createdAt" ON "reaction"("chatMessageId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_commentId_createdAt" ON "reaction"("commentId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_issueId_createdAt" ON "reaction"("issueId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_summary_resourceId_createdAt" ON "reaction_summary"("resourceId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_summary_chatMessageId_createdAt" ON "reaction_summary"("chatMessageId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_summary_commentId_createdAt" ON "reaction_summary"("commentId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reaction_summary_issueId_createdAt" ON "reaction_summary"("issueId", "createdAt");

-- CreateIndex
CREATE UNIQUE INDEX "reaction_summary_emoji_resourceId_chatMessageId_commentId_i_key" ON "reaction_summary"("emoji", "resourceId", "chatMessageId", "commentId", "issueId");

-- CreateIndex
CREATE INDEX "idx_reminder_list_userId_createdAt" ON "reminder_list"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reminder_reminderListId_createdAt" ON "reminder"("reminderListId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reminder_item_reminderId" ON "reminder_item"("reminderId");

-- CreateIndex
CREATE UNIQUE INDEX "report_publicId_key" ON "report"("publicId");

-- CreateIndex
CREATE INDEX "idx_report_publicId" ON "report"("publicId");

-- CreateIndex
CREATE INDEX "idx_report_createdById_createdAt" ON "report"("createdById", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_resourceVersionId_createdAt" ON "report"("resourceVersionId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_chatMessageId_createdAt" ON "report"("chatMessageId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_commentId_createdAt" ON "report"("commentId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_issueId_createdAt" ON "report"("issueId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_tagId_createdAt" ON "report"("tagId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_teamId_createdAt" ON "report"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_userId_createdAt" ON "report"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_response_reportId_createdAt" ON "report_response"("reportId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_report_response_createdById_createdAt" ON "report_response"("createdById", "createdAt");

-- CreateIndex
CREATE UNIQUE INDEX "report_response_reportId_createdById_key" ON "report_response"("reportId", "createdById");

-- CreateIndex
CREATE INDEX "idx_reputation_history_userId_createdAt" ON "reputation_history"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_reputation_history_userId_event" ON "reputation_history"("userId", "event");

-- CreateIndex
CREATE UNIQUE INDEX "resource_publicId_key" ON "resource"("publicId");

-- CreateIndex
CREATE INDEX "idx_resource_publicId" ON "resource"("publicId");

-- CreateIndex
CREATE INDEX "idx_resource_resourceType_createdAt_isDeleted_isInternal" ON "resource"("resourceType", "createdAt", "isDeleted", "isInternal");

-- CreateIndex
CREATE INDEX "idx_resource_resourceType_updatedAt_isDeleted_isInternal" ON "resource"("resourceType", "updatedAt", "isDeleted", "isInternal");

-- CreateIndex
CREATE INDEX "idx_resource_resourceType_score_isDeleted_isInternal" ON "resource"("resourceType", "score", "isDeleted", "isInternal");

-- CreateIndex
CREATE INDEX "idx_resource_resourceType_bookmarks_isDeleted_isInternal" ON "resource"("resourceType", "bookmarks", "isDeleted", "isInternal");

-- CreateIndex
CREATE INDEX "idx_resource_resourceType_views_isDeleted_isInternal" ON "resource"("resourceType", "views", "isDeleted", "isInternal");

-- CreateIndex
CREATE INDEX "idx_resource_createdById" ON "resource"("createdById");

-- CreateIndex
CREATE INDEX "idx_resource_ownedByTeamId" ON "resource"("ownedByTeamId");

-- CreateIndex
CREATE INDEX "idx_resource_ownedByUserId" ON "resource"("ownedByUserId");

-- CreateIndex
CREATE INDEX "idx_resource_parentId" ON "resource"("parentId");

-- CreateIndex
CREATE UNIQUE INDEX "resource_version_publicId_key" ON "resource_version"("publicId");

-- CreateIndex
CREATE UNIQUE INDEX "resource_version_pullRequestId_key" ON "resource_version"("pullRequestId");

-- CreateIndex
CREATE INDEX "idx_resource_version_rootId_createdAt" ON "resource_version"("rootId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_resource_version_rootId_updatedAt" ON "resource_version"("rootId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_resource_version_rootId_isLatest" ON "resource_version"("rootId", "isLatest");

-- CreateIndex
CREATE UNIQUE INDEX "resource_version_rootId_versionIndex_key" ON "resource_version"("rootId", "versionIndex");

-- CreateIndex
CREATE INDEX "idx_resource_version_relation_fromVersionId" ON "resource_version_relation"("fromVersionId");

-- CreateIndex
CREATE INDEX "idx_resource_version_relation_toVersionId" ON "resource_version_relation"("toVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "resource_version_relation_fromVersionId_toVersionId_relatio_key" ON "resource_version_relation"("fromVersionId", "toVersionId", "relationType");

-- CreateIndex
CREATE INDEX "idx_resource_translation_resourceVersionId" ON "resource_translation"("resourceVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "resource_translation_resourceVersionId_language_key" ON "resource_translation"("resourceVersionId", "language");

-- CreateIndex
CREATE INDEX "idx_resource_tag_taggedId" ON "resource_tag"("taggedId");

-- CreateIndex
CREATE INDEX "idx_resource_tag_tagTag" ON "resource_tag"("tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "resource_tag_taggedId_tagTag_key" ON "resource_tag"("taggedId", "tagTag");

-- CreateIndex
CREATE INDEX "idx_run_scheduleId" ON "run"("scheduleId");

-- CreateIndex
CREATE INDEX "idx_run_resourceVersionId_createdAt" ON "run"("resourceVersionId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_run_resourceVersionId_updatedAt" ON "run"("resourceVersionId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_run_teamId_createdAt" ON "run"("teamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_run_teamId_updatedAt" ON "run"("teamId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_run_userId_createdAt" ON "run"("userId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_run_userId_updatedAt" ON "run"("userId", "updatedAt");

-- CreateIndex
CREATE INDEX "idx_run_status" ON "run"("status");

-- CreateIndex
CREATE INDEX "idx_run_io_runId" ON "run_io"("runId");

-- CreateIndex
CREATE INDEX "idx_run_step_runId" ON "run_step"("runId");

-- CreateIndex
CREATE INDEX "idx_run_step_resourceVersionId" ON "run_step"("resourceVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "schedule_publicId_key" ON "schedule"("publicId");

-- CreateIndex
CREATE INDEX "schedule_publicId_idx" ON "schedule"("publicId");

-- CreateIndex
CREATE INDEX "idx_schedule_startTime" ON "schedule"("startTime");

-- CreateIndex
CREATE INDEX "idx_schedule_exception_scheduleId" ON "schedule_exception"("scheduleId");

-- CreateIndex
CREATE INDEX "idx_schedule_recurrence_scheduleId" ON "schedule_recurrence"("scheduleId");

-- CreateIndex
CREATE INDEX "idx_stats_resource_resourceId" ON "stats_resource"("resourceId");

-- CreateIndex
CREATE INDEX "idx_stats_resource_periodType_periodStart" ON "stats_resource"("periodType", "periodStart");

-- CreateIndex
CREATE UNIQUE INDEX "stats_resource_resourceId_periodStart_periodEnd_periodType_key" ON "stats_resource"("resourceId", "periodStart", "periodEnd", "periodType");

-- CreateIndex
CREATE INDEX "idx_stats_site_periodType_periodStart" ON "stats_site"("periodType", "periodStart");

-- CreateIndex
CREATE UNIQUE INDEX "stats_site_periodStart_periodEnd_periodType_key" ON "stats_site"("periodStart", "periodEnd", "periodType");

-- CreateIndex
CREATE INDEX "idx_stats_team_teamId" ON "stats_team"("teamId");

-- CreateIndex
CREATE INDEX "idx_stats_team_periodType_periodStart" ON "stats_team"("periodType", "periodStart");

-- CreateIndex
CREATE UNIQUE INDEX "stats_team_teamId_periodStart_periodEnd_periodType_key" ON "stats_team"("teamId", "periodStart", "periodEnd", "periodType");

-- CreateIndex
CREATE INDEX "idx_stats_user_userId" ON "stats_user"("userId");

-- CreateIndex
CREATE INDEX "idx_stats_user_periodType_periodStart" ON "stats_user"("periodType", "periodStart");

-- CreateIndex
CREATE UNIQUE INDEX "stats_user_userId_periodStart_periodEnd_periodType_key" ON "stats_user"("userId", "periodStart", "periodEnd", "periodType");

-- CreateIndex
CREATE UNIQUE INDEX "tag_tag_key" ON "tag"("tag");

-- CreateIndex
CREATE INDEX "idx_tag_createdById" ON "tag"("createdById");

-- CreateIndex
CREATE INDEX "idx_tag_translation_tagId" ON "tag_translation"("tagId");

-- CreateIndex
CREATE UNIQUE INDEX "tag_translation_tagId_language_key" ON "tag_translation"("tagId", "language");

-- CreateIndex
CREATE INDEX "idx_transfer_fromTeamId_createdAt" ON "transfer"("fromTeamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_transfer_fromUserId_createdAt" ON "transfer"("fromUserId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_transfer_toTeamId_createdAt" ON "transfer"("toTeamId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_transfer_toUserId_createdAt" ON "transfer"("toUserId", "createdAt");

-- CreateIndex
CREATE INDEX "idx_transfer_resourceId_createdAt" ON "transfer"("resourceId", "createdAt");

-- CreateIndex
CREATE UNIQUE INDEX "user_publicId_key" ON "user"("publicId");

-- CreateIndex
CREATE UNIQUE INDEX "user_confirmationCode_key" ON "user"("confirmationCode");

-- CreateIndex
CREATE UNIQUE INDEX "user_handle_key" ON "user"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "user_premiumId_key" ON "user"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "user_stripeCustomerId_key" ON "user"("stripeCustomerId");

-- CreateIndex
CREATE INDEX "idx_user_publicId" ON "user"("publicId");

-- CreateIndex
CREATE INDEX "idx_user_invitedByUserId_createdAt" ON "user"("invitedByUserId", "createdAt");

-- CreateIndex
CREATE UNIQUE INDEX "user_auth_resetPasswordCode_key" ON "user_auth"("resetPasswordCode");

-- CreateIndex
CREATE INDEX "idx_user_auth_user_id_createdAt" ON "user_auth"("user_id", "createdAt");

-- CreateIndex
CREATE UNIQUE INDEX "user_auth_provider_provider_user_id_key" ON "user_auth"("provider", "provider_user_id");

-- CreateIndex
CREATE INDEX "idx_session_user_id" ON "session"("user_id");

-- CreateIndex
CREATE INDEX "idx_session_auth_id" ON "session"("auth_id");

-- CreateIndex
CREATE INDEX "idx_user_translation_userId" ON "user_translation"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "user_translation_userId_language_key" ON "user_translation"("userId", "language");

-- CreateIndex
CREATE INDEX "idx_view_byId_lastViewedAt" ON "view"("byId", "lastViewedAt");

-- CreateIndex
CREATE INDEX "idx_view_issueId_lastViewedAt" ON "view"("issueId", "lastViewedAt");

-- CreateIndex
CREATE INDEX "idx_view_resourceId_lastViewedAt" ON "view"("resourceId", "lastViewedAt");

-- CreateIndex
CREATE INDEX "idx_view_teamId_lastViewedAt" ON "view"("teamId", "lastViewedAt");

-- CreateIndex
CREATE INDEX "idx_view_userId_lastViewedAt" ON "view"("userId", "lastViewedAt");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_stakingAddress_key" ON "wallet"("stakingAddress");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_publicAddress_key" ON "wallet"("publicAddress");

-- CreateIndex
CREATE INDEX "idx_wallet_teamId" ON "wallet"("teamId");

-- CreateIndex
CREATE INDEX "idx_wallet_userId" ON "wallet"("userId");

-- AddForeignKey
ALTER TABLE "award" ADD CONSTRAINT "award_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "chat_participants" ADD CONSTRAINT "chat_participants_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_participants" ADD CONSTRAINT "chat_participants_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_invite" ADD CONSTRAINT "chat_invite_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chat_invite" ADD CONSTRAINT "chat_invite_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_resourceVersionId_fkey" FOREIGN KEY ("resourceVersionId") REFERENCES "resource_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "issue" ADD CONSTRAINT "issue_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_closedById_fkey" FOREIGN KEY ("closedById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_translation" ADD CONSTRAINT "issue_translation_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "meeting_translation" ADD CONSTRAINT "meeting_translation_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification" ADD CONSTRAINT "notification_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_chatId_fkey" FOREIGN KEY ("chatId") REFERENCES "chat"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_reportId_fkey" FOREIGN KEY ("reportId") REFERENCES "report"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_subscriberId_fkey" FOREIGN KEY ("subscriberId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "push_device" ADD CONSTRAINT "push_device_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_premiumId_fkey" FOREIGN KEY ("premiumId") REFERENCES "premium"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team" ADD CONSTRAINT "team_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_translation" ADD CONSTRAINT "team_translation_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_tag" ADD CONSTRAINT "team_tag_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "team_tag" ADD CONSTRAINT "team_tag_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toResourceId_fkey" FOREIGN KEY ("toResourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request_translation" ADD CONSTRAINT "pull_request_translation_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction" ADD CONSTRAINT "reaction_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reaction_summary" ADD CONSTRAINT "reaction_summary_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder_list" ADD CONSTRAINT "reminder_list_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder" ADD CONSTRAINT "reminder_reminderListId_fkey" FOREIGN KEY ("reminderListId") REFERENCES "reminder_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder_item" ADD CONSTRAINT "reminder_item_reminderId_fkey" FOREIGN KEY ("reminderId") REFERENCES "reminder"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_resourceVersionId_fkey" FOREIGN KEY ("resourceVersionId") REFERENCES "resource_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_chatMessageId_fkey" FOREIGN KEY ("chatMessageId") REFERENCES "chat_message"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "resource" ADD CONSTRAINT "resource_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource" ADD CONSTRAINT "resource_ownedByTeamId_fkey" FOREIGN KEY ("ownedByTeamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource" ADD CONSTRAINT "resource_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource" ADD CONSTRAINT "resource_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "resource_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_version" ADD CONSTRAINT "resource_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_version" ADD CONSTRAINT "resource_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_version_relation" ADD CONSTRAINT "resource_version_relation_fromVersionId_fkey" FOREIGN KEY ("fromVersionId") REFERENCES "resource_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_version_relation" ADD CONSTRAINT "resource_version_relation_toVersionId_fkey" FOREIGN KEY ("toVersionId") REFERENCES "resource_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_translation" ADD CONSTRAINT "resource_translation_resourceVersionId_fkey" FOREIGN KEY ("resourceVersionId") REFERENCES "resource_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_tag" ADD CONSTRAINT "resource_tag_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "resource_tag" ADD CONSTRAINT "resource_tag_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_resourceVersionId_fkey" FOREIGN KEY ("resourceVersionId") REFERENCES "resource_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run" ADD CONSTRAINT "run_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_io" ADD CONSTRAINT "run_io_runId_fkey" FOREIGN KEY ("runId") REFERENCES "run"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_step" ADD CONSTRAINT "run_step_runId_fkey" FOREIGN KEY ("runId") REFERENCES "run"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_step" ADD CONSTRAINT "run_step_resourceVersionId_fkey" FOREIGN KEY ("resourceVersionId") REFERENCES "resource_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_exception" ADD CONSTRAINT "schedule_exception_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "schedule_recurrence" ADD CONSTRAINT "schedule_recurrence_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_resource" ADD CONSTRAINT "stats_resource_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "view" ADD CONSTRAINT "view_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_resourceId_fkey" FOREIGN KEY ("resourceId") REFERENCES "resource"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_teamId_fkey" FOREIGN KEY ("teamId") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- ------------------------------------------------------------
-- hnsw indexes for semantic-search columns
-- (run *after* the tables have been created)
-- ------------------------------------------------------------

CREATE INDEX chat_translation_embedding_hnsw
    ON "chat_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX issue_translation_embedding_hnsw
    ON "issue_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX meeting_translation_embedding_hnsw
    ON "meeting_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX team_translation_embedding_hnsw
    ON "team_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX reminder_embedding_hnsw
    ON "reminder"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX resource_translation_embedding_hnsw
    ON "resource_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX tag_translation_embedding_hnsw
    ON "tag_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);

CREATE INDEX user_translation_embedding_hnsw
    ON "user_translation"
    -- vector_l2_ops is Euclidean distance (you should see "<->" in the sql builder)
    USING hnsw (embedding vector_l2_ops)
    WITH (m = 16, ef_construction = 128);