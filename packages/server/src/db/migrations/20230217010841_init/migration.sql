CREATE EXTENSION IF NOT EXISTS citext;
-- CreateEnum
CREATE TYPE "AccountStatus" AS ENUM ('Deleted', 'Unlocked', 'SoftLocked', 'HardLocked');

-- CreateEnum
CREATE TYPE "AwardCategory" AS ENUM ('AccountAnniversary', 'AccountNew', 'ApiCreate', 'CommentCreate', 'IssueCreate', 'NoteCreate', 'ObjectBookmark', 'ObjectVote', 'OrganizationCreate', 'OrganizationJoin', 'PostCreate', 'ProjectCreate', 'PullRequestCreate', 'PullRequestComplete', 'QuestionAnswer', 'QuestionCreate', 'QuizPass', 'ReportEnd', 'ReportContribute', 'Reputation', 'RunRoutine', 'RunProject', 'RoutineCreate', 'SmartContractCreate', 'StandardCreate', 'Streak', 'UserInvite');

-- CreateEnum
CREATE TYPE "IssueStatus" AS ENUM ('Open', 'ClosedResolved', 'CloseUnresolved', 'Rejected');

-- CreateEnum
CREATE TYPE "MemberInviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "MeetingInviteStatus" AS ENUM ('Pending', 'Accepted', 'Declined');

-- CreateEnum
CREATE TYPE "NodeType" AS ENUM ('End', 'Redirect', 'RoutineList', 'Start');

-- CreateEnum
CREATE TYPE "PaymentStatus" AS ENUM ('Pending', 'Paid', 'Failed');

-- CreateEnum
CREATE TYPE "PeriodType" AS ENUM ('Hourly', 'Daily', 'Weekly', 'Monthly', 'Yearly');

-- CreateEnum
CREATE TYPE "PullRequestStatus" AS ENUM ('Open', 'Merged', 'Rejected');

-- CreateEnum
CREATE TYPE "QuizAttemptStatus" AS ENUM ('NotStarted', 'InProgress', 'Passed', 'Failed');

-- CreateEnum
CREATE TYPE "ReportStatus" AS ENUM ('ClosedDeleted', 'ClosedFalseReport', 'ClosedNonIssue', 'ClosedResolved', 'ClosedSuspended', 'Open');

-- CreateEnum
CREATE TYPE "ReportSuggestedAction" AS ENUM ('Delete', 'FalseReport', 'HideUntilFixed', 'NonIssue', 'SuspendUser');

-- CreateEnum
CREATE TYPE "ResourceUsedFor" AS ENUM ('Community', 'Context', 'Developer', 'Donation', 'ExternalService', 'Feed', 'Install', 'Learning', 'Notes', 'OfficialWebsite', 'Proposal', 'Related', 'Researching', 'Scheduling', 'Social', 'Tutorial');

-- CreateEnum
CREATE TYPE "RunStatus" AS ENUM ('Scheduled', 'InProgress', 'Completed', 'Failed', 'Cancelled');

-- CreateEnum
CREATE TYPE "RunStepStatus" AS ENUM ('InProgress', 'Completed', 'Skipped');

-- CreateEnum
CREATE TYPE "TransferStatus" AS ENUM ('Accepted', 'Denied', 'Pending');

-- CreateEnum
CREATE TYPE "UserScheduleFilterType" AS ENUM ('Blur', 'Hide', 'ShowMore');

-- CreateTable
CREATE TABLE "award" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "timeCurrentTierCompleted" TIMESTAMPTZ(6),
    "category" "AwardCategory" NOT NULL,
    "progress" INTEGER NOT NULL DEFAULT 0,
    "userId" UUID NOT NULL,

    CONSTRAINT "award_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,
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
CREATE TABLE "api_version" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "callLink" VARCHAR(1024) NOT NULL,
    "documentationLink" VARCHAR(1024),
    "isComplete" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
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
    "name" VARCHAR(128) NOT NULL,
    "summary" VARCHAR(1024),
    "details" VARCHAR(8192),
    "language" VARCHAR(3) NOT NULL,
    "apiVersionId" UUID NOT NULL,

    CONSTRAINT "api_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "api_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "api_key" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "key" VARCHAR(255) NOT NULL,
    "creditsUsedBeforeLimit" INTEGER NOT NULL DEFAULT 20000,
    "stopAtLimit" BOOLEAN NOT NULL DEFAULT true,
    "absoluteMax" INTEGER DEFAULT 1000000,
    "organizationId" UUID,
    "userId" UUID,

    CONSTRAINT "api_key_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "comment" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,
    "apiVersionId" UUID,
    "issueId" UUID,
    "noteVersionId" UUID,
    "parentId" UUID,
    "postId" UUID,
    "projectVersionId" UUID,
    "pullRequestId" UUID,
    "questionId" UUID,
    "questionAnswerId" UUID,
    "routineVersionId" UUID,
    "smartContractVersionId" UUID,
    "standardVersionId" UUID,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,

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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "emailAddress" CITEXT NOT NULL,
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
CREATE TABLE "issue" (
    "id" UUID NOT NULL,
    "status" "IssueStatus" NOT NULL DEFAULT 'Open',
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL,
    "closedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "apiId" UUID,
    "organizationId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,
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
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "issueId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "issue_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "label" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "label" VARCHAR(128) NOT NULL,
    "color" VARCHAR(7),
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,

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
CREATE TABLE "node" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "columnIndex" INTEGER,
    "rowIndex" INTEGER,
    "nodeType" "NodeType" NOT NULL,
    "runConditions" VARCHAR(4096),
    "voteConditions" VARCHAR(4096),
    "routineVersionId" UUID NOT NULL,

    CONSTRAINT "node_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL DEFAULT 'Name Me',
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
CREATE TABLE "node_end_next" (
    "id" UUID NOT NULL,
    "fromEndId" UUID NOT NULL,
    "toRoutineVersionId" UUID NOT NULL,

    CONSTRAINT "node_end_next_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_link" (
    "id" UUID NOT NULL,
    "fromId" UUID NOT NULL,
    "routineVersionId" UUID NOT NULL,
    "toId" UUID NOT NULL,

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
    "name" VARCHAR(128) NOT NULL,
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
    "name" VARCHAR(128) NOT NULL,
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
    "index" INTEGER NOT NULL,
    "isOptional" BOOLEAN NOT NULL DEFAULT false,
    "listId" UUID NOT NULL,
    "routineVersionId" UUID NOT NULL,

    CONSTRAINT "node_routine_list_item_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "node_routine_list_item_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128),
    "language" VARCHAR(3) NOT NULL,
    "itemId" UUID NOT NULL,

    CONSTRAINT "node_routine_list_item_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "note" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "parentId" UUID,
    "createdById" UUID,
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,
    "createdByOrganizationId" UUID,

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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT true,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
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
    "name" VARCHAR(128) NOT NULL,
    "text" VARCHAR(65536) NOT NULL,
    "description" VARCHAR(2048),
    "language" VARCHAR(3) NOT NULL,
    "noteVersionId" UUID NOT NULL,

    CONSTRAINT "note_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "notification" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "category" VARCHAR(64) NOT NULL,
    "isRead" BOOLEAN NOT NULL DEFAULT false,
    "title" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "apiId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "meetingId" UUID,
    "noteId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "pullRequestId" UUID,
    "questionId" UUID,
    "quizId" UUID,
    "reportId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,
    "subscriberId" UUID NOT NULL,
    "silent" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "notification_subscription_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "handle" VARCHAR(16),
    "isOpenToNewMembers" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "parentId" UUID,
    "premiumId" UUID,
    "resourceListId" UUID,
    "createdById" UUID,

    CONSTRAINT "organization_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "organization_language" (
    "id" UUID NOT NULL,
    "organizationId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "organization_language_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "inviteId" UUID,
    "openToAnyoneWithInvite" BOOLEAN NOT NULL DEFAULT false,
    "showOnOrganizationProfile" BOOLEAN NOT NULL DEFAULT false,
    "timeZone" VARCHAR(128),
    "eventStart" TIMESTAMPTZ(6),
    "eventEnd" TIMESTAMPTZ(6),
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "recurrStart" TIMESTAMPTZ(6),
    "recurrEnd" TIMESTAMPTZ(6),
    "organizationId" UUID NOT NULL,

    CONSTRAINT "meeting_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_attendees" (
    "id" UUID NOT NULL,
    "scheduleId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "meeting_attendees_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_invite" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "scheduleId" UUID NOT NULL,
    "roleId" UUID NOT NULL,

    CONSTRAINT "meeting_roles_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "meeting_translation" (
    "id" UUID NOT NULL,
    "scheduleId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "name" VARCHAR(128),
    "description" VARCHAR(2048),
    "link" VARCHAR(2048),

    CONSTRAINT "meeting_translation_pkey" PRIMARY KEY ("id")
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
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "organization_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isAdmin" BOOLEAN NOT NULL DEFAULT false,
    "permissions" VARCHAR(4096) NOT NULL,
    "organizationId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "member_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "member_invite" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "MemberInviteStatus" NOT NULL DEFAULT 'Pending',
    "message" VARCHAR(4096),
    "willBeAdmin" BOOLEAN NOT NULL DEFAULT false,
    "willHavePermissions" VARCHAR(4096),
    "organizationId" UUID NOT NULL,
    "userId" UUID NOT NULL,

    CONSTRAINT "member_invite_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "post" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "repostedFromId" UUID,
    "resourceListId" UUID,
    "isPinned" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "organizationId" UUID,
    "userId" UUID,

    CONSTRAINT "post_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "post_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "post_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "post_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "postId" UUID NOT NULL,

    CONSTRAINT "post_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "phone" (
    "id" UUID NOT NULL,
    "phoneNumber" VARCHAR(16) NOT NULL,
    "verified" BOOLEAN NOT NULL DEFAULT false,
    "lastVerifiedTime" TIMESTAMPTZ(6),
    "verificationCode" VARCHAR(6),
    "lastVerificationCodeRequestAttempt" TIMESTAMPTZ(6),
    "organizationId" UUID,
    "userId" UUID,

    CONSTRAINT "phone_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "payment" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "amount" INTEGER NOT NULL,
    "currency" VARCHAR(255) NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "paymentMethod" VARCHAR(255) NOT NULL,
    "status" "PaymentStatus" NOT NULL DEFAULT 'Pending',
    "organizationId" UUID,
    "userId" UUID,
    "cardType" VARCHAR(255),
    "cardExpDate" VARCHAR(255),
    "cardLast4" VARCHAR(255),

    CONSTRAINT "payment_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "premium" (
    "id" UUID NOT NULL,
    "customPlan" VARCHAR(2048),
    "enabledAt" TIMESTAMPTZ(6),
    "expiresAt" TIMESTAMPTZ(6),
    "isActive" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "premium_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "handle" VARCHAR(16),
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,
    "parentId" UUID,

    CONSTRAINT "project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "project_version" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "complexity" INTEGER NOT NULL DEFAULT 1,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "isComplete" BOOLEAN NOT NULL DEFAULT true,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "PullRequestStatus" NOT NULL DEFAULT 'Open',
    "mergedOrRejectedAt" TIMESTAMPTZ(6),
    "createdById" UUID,
    "toApiId" UUID,
    "toNoteId" UUID,
    "toProjectId" UUID,
    "toRoutineId" UUID,
    "toSmartContractId" UUID,
    "toStandardId" UUID,

    CONSTRAINT "pull_request_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "question" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "referencing" VARCHAR(2048),
    "hasAcceptedAnswer" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "apiId" UUID,
    "noteId" UUID,
    "organizationId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,
    "createdById" UUID,

    CONSTRAINT "question_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "question_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "questionId" UUID NOT NULL,

    CONSTRAINT "question_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "question_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "question_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "question_answer" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "isAccepted" BOOLEAN NOT NULL DEFAULT false,
    "questionId" UUID NOT NULL,
    "createdById" UUID,

    CONSTRAINT "question_answer_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "question_answer_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "answerId" UUID NOT NULL,

    CONSTRAINT "question_answer_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "maxAttempts" INTEGER,
    "randomizeQuestionOrder" BOOLEAN NOT NULL DEFAULT false,
    "revealCorrectAnswers" BOOLEAN NOT NULL DEFAULT true,
    "timeLimit" INTEGER,
    "wasAutoGenerated" BOOLEAN NOT NULL DEFAULT false,
    "pointsToPass" INTEGER,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "routineId" UUID,
    "projectId" UUID,
    "createdById" UUID,

    CONSTRAINT "quiz_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "quizId" UUID NOT NULL,

    CONSTRAINT "quiz_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_attempt" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "pointsEarned" INTEGER NOT NULL DEFAULT 0,
    "language" VARCHAR(3) NOT NULL,
    "status" "QuizAttemptStatus" NOT NULL DEFAULT 'NotStarted',
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "timeTaken" INTEGER,
    "quizId" UUID NOT NULL,
    "userId" UUID,

    CONSTRAINT "quiz_attempt_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_question_response" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "quizAttemptId" UUID NOT NULL,
    "quizQuestionId" UUID NOT NULL,

    CONSTRAINT "quiz_question_response_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_question_response_translation" (
    "id" UUID NOT NULL,
    "response" VARCHAR(8192) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "responseId" UUID NOT NULL,

    CONSTRAINT "quiz_question_response_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_question" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "order" INTEGER,
    "points" INTEGER NOT NULL DEFAULT 1,
    "standardVersionId" UUID,
    "quizId" UUID NOT NULL,

    CONSTRAINT "quiz_question_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "quiz_question_translation" (
    "id" UUID NOT NULL,
    "helpText" VARCHAR(2048),
    "questionText" VARCHAR(1024) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "questionId" UUID NOT NULL,

    CONSTRAINT "quiz_question_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_list" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "reminder_list_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "dueDate" TIMESTAMPTZ(6),
    "completed" BOOLEAN NOT NULL DEFAULT false,
    "index" INTEGER NOT NULL,
    "reminderListId" UUID NOT NULL,

    CONSTRAINT "reminder_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "reminder_item" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "dueDate" TIMESTAMPTZ(6),
    "index" INTEGER NOT NULL,
    "reminderId" UUID NOT NULL,
    "completed" BOOLEAN NOT NULL DEFAULT false,

    CONSTRAINT "reminder_item_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report" (
    "id" UUID NOT NULL,
    "reason" VARCHAR(128) NOT NULL,
    "details" VARCHAR(8192),
    "language" VARCHAR(3) NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "ReportStatus" NOT NULL,
    "apiVersionId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteVersionId" UUID,
    "organizationId" UUID,
    "postId" UUID,
    "projectVersionId" UUID,
    "questionId" UUID,
    "routineVersionId" UUID,
    "smartContractVersionId" UUID,
    "standardVersionId" UUID,
    "tagId" UUID,
    "userId" UUID,
    "createdById" UUID,

    CONSTRAINT "report_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "report_response" (
    "id" UUID NOT NULL,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,

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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "permissions" VARCHAR(4096) NOT NULL,
    "organizationId" UUID NOT NULL,

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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "ownedByOrganizationId" UUID,
    "parentId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "apiCallData" VARCHAR(8192),
    "complexity" INTEGER NOT NULL DEFAULT 1,
    "intendToPullRequest" BOOLEAN NOT NULL DEFAULT false,
    "isAutomatable" BOOLEAN NOT NULL DEFAULT false,
    "isComplete" BOOLEAN NOT NULL DEFAULT true,
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "rootId" UUID NOT NULL,
    "simplicity" INTEGER NOT NULL DEFAULT 1,
    "timesStarted" INTEGER NOT NULL DEFAULT 0,
    "timesCompleted" INTEGER NOT NULL DEFAULT 0,
    "smartContractCallData" VARCHAR(8192),
    "resourceListId" UUID,
    "apiVersionId" UUID,
    "smartContractVersionId" UUID,
    "pullRequestId" UUID,
    "versionIndex" INTEGER NOT NULL DEFAULT 0,
    "versionLabel" VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    "versionNotes" VARCHAR(4096),

    CONSTRAINT "routine_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "routine_version_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "instructions" VARCHAR(8192) NOT NULL,
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
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "scheduleId" UUID,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT 'Scheduled',
    "projectVersionId" UUID,
    "userId" UUID,
    "organizationId" UUID,

    CONSTRAINT "run_project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project_step" (
    "id" UUID NOT NULL,
    "order" INTEGER NOT NULL,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "runProjectId" UUID NOT NULL,
    "directoryId" UUID,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "step" INTEGER[],
    "status" "RunStepStatus" NOT NULL DEFAULT 'InProgress',
    "name" VARCHAR(128) NOT NULL,

    CONSTRAINT "run_project_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project_schedule" (
    "id" UUID NOT NULL,
    "timeZone" VARCHAR(128),
    "windowStart" TIMESTAMPTZ(6),
    "windowEnd" TIMESTAMPTZ(6),
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "recurrStart" TIMESTAMPTZ(6),
    "recurrEnd" TIMESTAMPTZ(6),

    CONSTRAINT "run_project_schedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project_schedule_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "run_project_schedule_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_project_schedule_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "runProjectScheduleId" UUID NOT NULL,

    CONSTRAINT "run_project_schedule_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedComplexity" INTEGER NOT NULL DEFAULT 0,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "scheduleId" UUID,
    "wasRunAutomatically" BOOLEAN NOT NULL DEFAULT false,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "name" VARCHAR(128) NOT NULL,
    "status" "RunStatus" NOT NULL DEFAULT 'Scheduled',
    "routineVersionId" UUID,
    "userId" UUID,
    "organizationId" UUID,
    "runProjectId" UUID,

    CONSTRAINT "run_routine_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_input" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "data" VARCHAR(8192) NOT NULL,
    "inputId" UUID NOT NULL,
    "runRoutineId" UUID NOT NULL,

    CONSTRAINT "run_routine_input_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_step" (
    "id" UUID NOT NULL,
    "order" INTEGER NOT NULL,
    "contextSwitches" INTEGER NOT NULL DEFAULT 0,
    "runRoutineId" UUID NOT NULL,
    "nodeId" UUID,
    "subroutineVersionId" UUID,
    "startedAt" TIMESTAMPTZ(6),
    "timeElapsed" INTEGER,
    "completedAt" TIMESTAMPTZ(6),
    "step" INTEGER[],
    "status" "RunStepStatus" NOT NULL DEFAULT 'InProgress',
    "name" VARCHAR(128) NOT NULL,

    CONSTRAINT "run_routine_step_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_schedule" (
    "id" UUID NOT NULL,
    "attemptAutomatic" BOOLEAN NOT NULL DEFAULT true,
    "maxAutomaticAttempts" INTEGER NOT NULL DEFAULT 2,
    "timeZone" VARCHAR(128),
    "windowStart" TIMESTAMPTZ(6),
    "windowEnd" TIMESTAMPTZ(6),
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "recurrStart" TIMESTAMPTZ(6),
    "recurrEnd" TIMESTAMPTZ(6),

    CONSTRAINT "run_routine_schedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_schedule_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "run_routine_schedule_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "run_routine_schedule_translation" (
    "id" UUID NOT NULL,
    "description" VARCHAR(2048),
    "name" VARCHAR(128) NOT NULL,
    "language" VARCHAR(3) NOT NULL,
    "runRoutineScheduleId" UUID NOT NULL,

    CONSTRAINT "run_routine_schedule_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "smart_contract" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "hasCompleteVersion" BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMPTZ(6),
    "isDeleted" BOOLEAN NOT NULL DEFAULT false,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "score" INTEGER NOT NULL DEFAULT 0,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "permissions" VARCHAR(4096) NOT NULL,
    "createdById" UUID,
    "ownedByOrganizationId" UUID,
    "parentId" UUID,
    "ownedByUserId" UUID,

    CONSTRAINT "smart_contract_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "smart_contract_version" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "default" VARCHAR(2048),
    "contractType" VARCHAR(256) NOT NULL,
    "content" VARCHAR(8192) NOT NULL,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
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

    CONSTRAINT "smart_contract_version_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "smart_contract_version_translation" (
    "id" UUID NOT NULL,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "jsonVariable" VARCHAR(8192),
    "smartContractVersionId" UUID NOT NULL,
    "language" VARCHAR(3) NOT NULL,

    CONSTRAINT "smart_contract_version_translation_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "smart_contract_tags" (
    "id" UUID NOT NULL,
    "taggedId" UUID NOT NULL,
    "tagTag" VARCHAR(128) NOT NULL,

    CONSTRAINT "smart_contract_tags_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "smart_contract_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "smart_contract_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    "ownedByUserId" UUID,
    "ownedByOrganizationId" UUID,

    CONSTRAINT "standard_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "standard_version" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMPTZ(6),
    "default" VARCHAR(2048),
    "standardType" TEXT NOT NULL,
    "props" VARCHAR(8192) NOT NULL,
    "isLatest" BOOLEAN NOT NULL DEFAULT false,
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
CREATE TABLE "bookmark" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "label" VARCHAR(128) DEFAULT 'Favorites',
    "byId" UUID NOT NULL,
    "apiId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "organizationId" UUID,
    "postId" UUID,
    "projectId" UUID,
    "questionId" UUID,
    "questionAnswerId" UUID,
    "quizId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,
    "tagId" UUID,
    "userId" UUID,

    CONSTRAINT "bookmark_pkey" PRIMARY KEY ("id")
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
    "organizationsCreated" INTEGER NOT NULL,
    "projectsCreated" INTEGER NOT NULL,
    "projectsCompleted" INTEGER NOT NULL,
    "projectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "quizzesCreated" INTEGER NOT NULL,
    "quizzesCompleted" INTEGER NOT NULL,
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
    "smartContractsCreated" INTEGER NOT NULL,
    "smartContractsCompleted" INTEGER NOT NULL,
    "smartContractCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "smartContractCalls" INTEGER NOT NULL,
    "standardsCreated" INTEGER NOT NULL,
    "standardsCompleted" INTEGER NOT NULL,
    "standardCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
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
CREATE TABLE "stats_organization" (
    "id" UUID NOT NULL,
    "organizationId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "apis" INTEGER NOT NULL,
    "members" INTEGER NOT NULL,
    "notes" INTEGER NOT NULL,
    "projects" INTEGER NOT NULL,
    "routines" INTEGER NOT NULL,
    "runRoutinesStarted" INTEGER NOT NULL,
    "runRoutinesCompleted" INTEGER NOT NULL,
    "runRoutineCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runRoutineContextSwitchesAverage" DOUBLE PRECISION NOT NULL,
    "smartContracts" INTEGER NOT NULL,
    "standards" INTEGER NOT NULL,

    CONSTRAINT "stats_organization_pkey" PRIMARY KEY ("id")
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
    "notes" INTEGER NOT NULL,
    "organizations" INTEGER NOT NULL,
    "projects" INTEGER NOT NULL,
    "routines" INTEGER NOT NULL,
    "smartContracts" INTEGER NOT NULL,
    "standards" INTEGER NOT NULL,
    "runsStarted" INTEGER NOT NULL,
    "runsCompleted" INTEGER NOT NULL,
    "runCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "runContextSwitchesAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_project_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "stats_quiz" (
    "id" UUID NOT NULL,
    "quizId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "timesStarted" INTEGER NOT NULL,
    "timesPassed" INTEGER NOT NULL,
    "timesFailed" INTEGER NOT NULL,
    "scoreAverage" DOUBLE PRECISION NOT NULL,
    "completionTimeAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_quiz_pkey" PRIMARY KEY ("id")
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
CREATE TABLE "stats_smart_contract" (
    "id" UUID NOT NULL,
    "smartContractId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "calls" INTEGER NOT NULL,
    "routineVersions" INTEGER NOT NULL,

    CONSTRAINT "stats_smart_contract_pkey" PRIMARY KEY ("id")
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
CREATE TABLE "stats_user" (
    "id" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "periodStart" TIMESTAMPTZ(6) NOT NULL,
    "periodEnd" TIMESTAMPTZ(6) NOT NULL,
    "periodType" "PeriodType" NOT NULL,
    "apisCreated" INTEGER NOT NULL,
    "organizationsCreated" INTEGER NOT NULL,
    "projectsCreated" INTEGER NOT NULL,
    "projectsCompleted" INTEGER NOT NULL,
    "projectCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "quizzesPassed" INTEGER NOT NULL,
    "quizzesFailed" INTEGER NOT NULL,
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
    "smartContractsCreated" INTEGER NOT NULL,
    "smartContractsCompleted" INTEGER NOT NULL,
    "smartContractCompletionTimeAverage" DOUBLE PRECISION NOT NULL,
    "standardsCreated" INTEGER NOT NULL,
    "standardsCompleted" INTEGER NOT NULL,
    "standardCompletionTimeAverage" DOUBLE PRECISION NOT NULL,

    CONSTRAINT "stats_user_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "tag" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "tag" VARCHAR(128) NOT NULL,
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "createdById" UUID,

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
CREATE TABLE "transfer" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "TransferStatus" NOT NULL DEFAULT 'Pending',
    "initializedByReceiver" BOOLEAN NOT NULL DEFAULT false,
    "message" VARCHAR(4096),
    "denyReason" VARCHAR(2048),
    "fromUserId" UUID,
    "fromOrganizationId" UUID,
    "toUserId" UUID,
    "toOrganizationId" UUID,
    "apiId" UUID,
    "noteId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,

    CONSTRAINT "transfer_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "confirmationCode" VARCHAR(256),
    "confirmationCodeDate" TIMESTAMPTZ(6),
    "invitedByUserId" UUID,
    "isPrivate" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateApis" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateApisCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateMemberships" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateOrganizationsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateProjects" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateProjectsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivatePullRequests" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateQuestionsAnswered" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateQuestionsAsked" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateQuizzesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoles" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoutines" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateRoutinesCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateSmartContracts" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateSmartContractsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateStandards" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateStandardsCreated" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateBookmarks" BOOLEAN NOT NULL DEFAULT false,
    "isPrivateVotes" BOOLEAN NOT NULL DEFAULT false,
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
    "theme" VARCHAR(255) NOT NULL DEFAULT 'light',
    "handle" VARCHAR(16),
    "currentStreak" INTEGER NOT NULL DEFAULT 0,
    "longestStreak" INTEGER NOT NULL DEFAULT 0,
    "accountTabsOrder" VARCHAR(255),
    "notificationSettings" VARCHAR(2048),
    "bookmarks" INTEGER NOT NULL DEFAULT 0,
    "views" INTEGER NOT NULL DEFAULT 0,
    "premiumId" UUID,
    "status" "AccountStatus" NOT NULL DEFAULT 'Unlocked',

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
CREATE TABLE "user_schedule" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "description" VARCHAR(2048),
    "timeZone" VARCHAR(128),
    "eventStart" TIMESTAMPTZ(6),
    "eventEnd" TIMESTAMPTZ(6),
    "recurring" BOOLEAN NOT NULL DEFAULT false,
    "recurrStart" TIMESTAMPTZ(6),
    "recurrEnd" TIMESTAMPTZ(6),
    "reminderListId" UUID,
    "resourceListId" UUID,
    "userId" UUID NOT NULL,

    CONSTRAINT "user_schedule_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_schedule_labels" (
    "id" UUID NOT NULL,
    "labelledId" UUID NOT NULL,
    "labelId" UUID NOT NULL,

    CONSTRAINT "user_schedule_labels_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "user_schedule_filter" (
    "id" UUID NOT NULL,
    "filterType" "UserScheduleFilterType" NOT NULL,
    "userScheduleId" UUID NOT NULL,
    "tagId" UUID NOT NULL,

    CONSTRAINT "user_schedule_filter_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "view" (
    "id" UUID NOT NULL,
    "lastViewedAt" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "name" VARCHAR(128) NOT NULL,
    "byId" UUID NOT NULL,
    "apiId" UUID,
    "issueId" UUID,
    "organizationId" UUID,
    "questionId" UUID,
    "noteId" UUID,
    "postId" UUID,
    "projectId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,
    "userId" UUID,

    CONSTRAINT "view_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "vote" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "isUpvote" BOOLEAN NOT NULL DEFAULT true,
    "byId" UUID NOT NULL,
    "apiId" UUID,
    "commentId" UUID,
    "issueId" UUID,
    "noteId" UUID,
    "postId" UUID,
    "projectId" UUID,
    "questionId" UUID,
    "questionAnswerId" UUID,
    "quizId" UUID,
    "routineId" UUID,
    "smartContractId" UUID,
    "standardId" UUID,

    CONSTRAINT "vote_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "wallet" (
    "id" UUID NOT NULL,
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
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

    CONSTRAINT "wallet_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "_api_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_note_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_organizationToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_memberTorole" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_project_version_directory_listing" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_project_version_directoryToroutine_version" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_project_version_directoryTosmart_contract_version" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateTable
CREATE TABLE "_project_version_directoryTostandard_version" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- CreateIndex
CREATE UNIQUE INDEX "award_userId_category_key" ON "award"("userId", "category");

-- CreateIndex
CREATE UNIQUE INDEX "api_labels_labelledId_labelId_key" ON "api_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_callLink_key" ON "api_version"("callLink");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_resourceListId_key" ON "api_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_pullRequestId_key" ON "api_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "api_version_rootId_versionIndex_key" ON "api_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "api_tags_taggedId_tagTag_key" ON "api_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "api_key_key_key" ON "api_key"("key");

-- CreateIndex
CREATE UNIQUE INDEX "comment_translation_commentId_language_key" ON "comment_translation"("commentId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "email_emailAddress_key" ON "email"("emailAddress");

-- CreateIndex
CREATE UNIQUE INDEX "email_verificationCode_key" ON "email"("verificationCode");

-- CreateIndex
CREATE UNIQUE INDEX "handle_handle_key" ON "handle"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "issue_labels_labelledId_labelId_key" ON "issue_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "issue_translation_issueId_language_key" ON "issue_translation"("issueId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "label_label_key" ON "label"("label");

-- CreateIndex
CREATE UNIQUE INDEX "label_label_ownedByUserId_ownedByOrganizationId_key" ON "label"("label", "ownedByUserId", "ownedByOrganizationId");

-- CreateIndex
CREATE UNIQUE INDEX "label_translation_labelId_language_key" ON "label_translation"("labelId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "node_translation_nodeId_language_key" ON "node_translation"("nodeId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "node_end_nodeId_key" ON "node_end"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_end_next_fromEndId_toRoutineVersionId_key" ON "node_end_next"("fromEndId", "toRoutineVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "node_link_when_translation_whenId_language_key" ON "node_link_when_translation"("whenId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "node_loop_nodeId_key" ON "node_loop"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_nodeId_key" ON "node_routine_list"("nodeId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_item_listId_routineVersionId_key" ON "node_routine_list_item"("listId", "routineVersionId");

-- CreateIndex
CREATE UNIQUE INDEX "node_routine_list_item_translation_itemId_language_key" ON "node_routine_list_item_translation"("itemId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "note_labels_labelledId_labelId_key" ON "note_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "note_tags_taggedId_tagTag_key" ON "note_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "note_version_pullRequestId_key" ON "note_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "note_version_rootId_versionIndex_key" ON "note_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "push_device_endpoint_key" ON "push_device"("endpoint");

-- CreateIndex
CREATE UNIQUE INDEX "organization_handle_key" ON "organization"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "organization_premiumId_key" ON "organization"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "organization_resourceListId_key" ON "organization"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "organization_language_organizationId_language_key" ON "organization_language"("organizationId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_attendees_scheduleId_userId_key" ON "meeting_attendees"("scheduleId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_invite_meetingId_userId_key" ON "meeting_invite"("meetingId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_labels_labelledId_labelId_key" ON "meeting_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_roles_scheduleId_roleId_key" ON "meeting_roles"("scheduleId", "roleId");

-- CreateIndex
CREATE UNIQUE INDEX "meeting_translation_scheduleId_language_key" ON "meeting_translation"("scheduleId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "organization_translation_organizationId_language_key" ON "organization_translation"("organizationId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "organization_tags_taggedId_tagTag_key" ON "organization_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "member_organizationId_userId_key" ON "member"("organizationId", "userId");

-- CreateIndex
CREATE UNIQUE INDEX "member_invite_userId_key" ON "member_invite"("userId");

-- CreateIndex
CREATE UNIQUE INDEX "member_invite_userId_organizationId_key" ON "member_invite"("userId", "organizationId");

-- CreateIndex
CREATE UNIQUE INDEX "post_resourceListId_key" ON "post"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "post_tags_taggedId_tagTag_key" ON "post_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "post_translation_postId_language_key" ON "post_translation"("postId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "phone_phoneNumber_key" ON "phone"("phoneNumber");

-- CreateIndex
CREATE UNIQUE INDEX "phone_verificationCode_key" ON "phone"("verificationCode");

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
CREATE UNIQUE INDEX "question_translation_questionId_language_key" ON "question_translation"("questionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "question_tags_taggedId_tagTag_key" ON "question_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "question_answer_translation_answerId_language_key" ON "question_answer_translation"("answerId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "quiz_translation_quizId_language_key" ON "quiz_translation"("quizId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "quiz_question_response_quizAttemptId_quizQuestionId_key" ON "quiz_question_response"("quizAttemptId", "quizQuestionId");

-- CreateIndex
CREATE UNIQUE INDEX "quiz_question_response_translation_responseId_language_key" ON "quiz_question_response_translation"("responseId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "quiz_question_translation_questionId_language_key" ON "quiz_question_translation"("questionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "report_response_reportId_createdById_key" ON "report_response"("reportId", "createdById");

-- CreateIndex
CREATE UNIQUE INDEX "resource_translation_resourceId_language_key" ON "resource_translation"("resourceId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "resource_list_translation_listId_language_key" ON "resource_list_translation"("listId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "role_organizationId_name_key" ON "role"("organizationId", "name");

-- CreateIndex
CREATE UNIQUE INDEX "role_translation_roleId_language_key" ON "role_translation"("roleId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_resourceListId_key" ON "routine_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_pullRequestId_key" ON "routine_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "routine_version_rootId_versionIndex_key" ON "routine_version"("rootId", "versionIndex");

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
CREATE UNIQUE INDEX "run_project_schedule_labels_labelledId_labelId_key" ON "run_project_schedule_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "run_project_schedule_translation_runProjectScheduleId_langu_key" ON "run_project_schedule_translation"("runProjectScheduleId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "run_routine_schedule_labels_labelledId_labelId_key" ON "run_routine_schedule_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "run_routine_schedule_translation_runRoutineScheduleId_langu_key" ON "run_routine_schedule_translation"("runRoutineScheduleId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_version_resourceListId_key" ON "smart_contract_version"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_version_pullRequestId_key" ON "smart_contract_version"("pullRequestId");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_version_rootId_versionIndex_key" ON "smart_contract_version"("rootId", "versionIndex");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_version_translation_smartContractVersionId_l_key" ON "smart_contract_version_translation"("smartContractVersionId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_tags_taggedId_tagTag_key" ON "smart_contract_tags"("taggedId", "tagTag");

-- CreateIndex
CREATE UNIQUE INDEX "smart_contract_labels_labelledId_labelId_key" ON "smart_contract_labels"("labelledId", "labelId");

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
CREATE UNIQUE INDEX "user_resetPasswordCode_key" ON "user"("resetPasswordCode");

-- CreateIndex
CREATE UNIQUE INDEX "user_handle_key" ON "user"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "user_premiumId_key" ON "user"("premiumId");

-- CreateIndex
CREATE UNIQUE INDEX "user_translation_userId_language_key" ON "user_translation"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_language_userId_language_key" ON "user_language"("userId", "language");

-- CreateIndex
CREATE UNIQUE INDEX "user_schedule_reminderListId_key" ON "user_schedule"("reminderListId");

-- CreateIndex
CREATE UNIQUE INDEX "user_schedule_resourceListId_key" ON "user_schedule"("resourceListId");

-- CreateIndex
CREATE UNIQUE INDEX "user_schedule_labels_labelledId_labelId_key" ON "user_schedule_labels"("labelledId", "labelId");

-- CreateIndex
CREATE UNIQUE INDEX "user_schedule_filter_userScheduleId_tagId_key" ON "user_schedule_filter"("userScheduleId", "tagId");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_stakingAddress_key" ON "wallet"("stakingAddress");

-- CreateIndex
CREATE UNIQUE INDEX "wallet_publicAddress_key" ON "wallet"("publicAddress");

-- CreateIndex
CREATE UNIQUE INDEX "_api_versionToproject_version_directory_AB_unique" ON "_api_versionToproject_version_directory"("A", "B");

-- CreateIndex
CREATE INDEX "_api_versionToproject_version_directory_B_index" ON "_api_versionToproject_version_directory"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_note_versionToproject_version_directory_AB_unique" ON "_note_versionToproject_version_directory"("A", "B");

-- CreateIndex
CREATE INDEX "_note_versionToproject_version_directory_B_index" ON "_note_versionToproject_version_directory"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_organizationToproject_version_directory_AB_unique" ON "_organizationToproject_version_directory"("A", "B");

-- CreateIndex
CREATE INDEX "_organizationToproject_version_directory_B_index" ON "_organizationToproject_version_directory"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_memberTorole_AB_unique" ON "_memberTorole"("A", "B");

-- CreateIndex
CREATE INDEX "_memberTorole_B_index" ON "_memberTorole"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_project_version_directory_listing_AB_unique" ON "_project_version_directory_listing"("A", "B");

-- CreateIndex
CREATE INDEX "_project_version_directory_listing_B_index" ON "_project_version_directory_listing"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_project_version_directoryToroutine_version_AB_unique" ON "_project_version_directoryToroutine_version"("A", "B");

-- CreateIndex
CREATE INDEX "_project_version_directoryToroutine_version_B_index" ON "_project_version_directoryToroutine_version"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_project_version_directoryTosmart_contract_version_AB_unique" ON "_project_version_directoryTosmart_contract_version"("A", "B");

-- CreateIndex
CREATE INDEX "_project_version_directoryTosmart_contract_version_B_index" ON "_project_version_directoryTosmart_contract_version"("B");

-- CreateIndex
CREATE UNIQUE INDEX "_project_version_directoryTostandard_version_AB_unique" ON "_project_version_directoryTostandard_version"("A", "B");

-- CreateIndex
CREATE INDEX "_project_version_directoryTostandard_version_B_index" ON "_project_version_directoryTostandard_version"("B");

-- AddForeignKey
ALTER TABLE "award" ADD CONSTRAINT "award_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api" ADD CONSTRAINT "api_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "api_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_labels" ADD CONSTRAINT "api_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_labels" ADD CONSTRAINT "api_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version" ADD CONSTRAINT "api_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_version_translation" ADD CONSTRAINT "api_version_translation_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_tags" ADD CONSTRAINT "api_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_tags" ADD CONSTRAINT "api_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key" ADD CONSTRAINT "api_key_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "api_key" ADD CONSTRAINT "api_key_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_noteVersionId_fkey" FOREIGN KEY ("noteVersionId") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_questionAnswerId_fkey" FOREIGN KEY ("questionAnswerId") REFERENCES "question_answer"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_smartContractVersionId_fkey" FOREIGN KEY ("smartContractVersionId") REFERENCES "smart_contract_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment" ADD CONSTRAINT "comment_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "comment_translation" ADD CONSTRAINT "comment_translation_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "email" ADD CONSTRAINT "email_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "handle" ADD CONSTRAINT "handle_walletId_fkey" FOREIGN KEY ("walletId") REFERENCES "wallet"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_closedById_fkey" FOREIGN KEY ("closedById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue" ADD CONSTRAINT "issue_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_labels" ADD CONSTRAINT "issue_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_labels" ADD CONSTRAINT "issue_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "issue_translation" ADD CONSTRAINT "issue_translation_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label" ADD CONSTRAINT "label_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label" ADD CONSTRAINT "label_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "label_translation" ADD CONSTRAINT "label_translation_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node" ADD CONSTRAINT "node_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_translation" ADD CONSTRAINT "node_translation_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_end" ADD CONSTRAINT "node_end_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_end_next" ADD CONSTRAINT "node_end_next_fromEndId_fkey" FOREIGN KEY ("fromEndId") REFERENCES "node_end"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_end_next" ADD CONSTRAINT "node_end_next_toRoutineVersionId_fkey" FOREIGN KEY ("toRoutineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_fromId_fkey" FOREIGN KEY ("fromId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_link" ADD CONSTRAINT "node_link_toId_fkey" FOREIGN KEY ("toId") REFERENCES "node"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "node_routine_list_item" ADD CONSTRAINT "node_routine_list_item_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "node_routine_list_item_translation" ADD CONSTRAINT "node_routine_list_item_translation_itemId_fkey" FOREIGN KEY ("itemId") REFERENCES "node_routine_list_item"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "note_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "note" ADD CONSTRAINT "note_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "notification" ADD CONSTRAINT "notification_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "push_device" ADD CONSTRAINT "push_device_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_meetingId_fkey" FOREIGN KEY ("meetingId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_reportId_fkey" FOREIGN KEY ("reportId") REFERENCES "report"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "notification_subscription" ADD CONSTRAINT "notification_subscription_subscriberId_fkey" FOREIGN KEY ("subscriberId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization" ADD CONSTRAINT "organization_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization" ADD CONSTRAINT "organization_premiumId_fkey" FOREIGN KEY ("premiumId") REFERENCES "premium"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization" ADD CONSTRAINT "organization_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization" ADD CONSTRAINT "organization_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_language" ADD CONSTRAINT "organization_language_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting" ADD CONSTRAINT "meeting_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_attendees" ADD CONSTRAINT "meeting_attendees_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "meeting_roles" ADD CONSTRAINT "meeting_roles_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_roles" ADD CONSTRAINT "meeting_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "meeting_translation" ADD CONSTRAINT "meeting_translation_scheduleId_fkey" FOREIGN KEY ("scheduleId") REFERENCES "meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_translation" ADD CONSTRAINT "organization_translation_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_tags" ADD CONSTRAINT "organization_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "organization_tags" ADD CONSTRAINT "organization_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member" ADD CONSTRAINT "member_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member" ADD CONSTRAINT "member_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member_invite" ADD CONSTRAINT "member_invite_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "member_invite" ADD CONSTRAINT "member_invite_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post" ADD CONSTRAINT "post_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post" ADD CONSTRAINT "post_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post" ADD CONSTRAINT "post_repostedFromId_fkey" FOREIGN KEY ("repostedFromId") REFERENCES "post"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post" ADD CONSTRAINT "post_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post_tags" ADD CONSTRAINT "post_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post_tags" ADD CONSTRAINT "post_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "post_translation" ADD CONSTRAINT "post_translation_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "phone" ADD CONSTRAINT "phone_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "phone" ADD CONSTRAINT "phone_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "payment_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "payment" ADD CONSTRAINT "payment_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "project" ADD CONSTRAINT "project_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

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
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toNoteId_fkey" FOREIGN KEY ("toNoteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toProjectId_fkey" FOREIGN KEY ("toProjectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toRoutineId_fkey" FOREIGN KEY ("toRoutineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toSmartContractId_fkey" FOREIGN KEY ("toSmartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "pull_request" ADD CONSTRAINT "pull_request_toStandardId_fkey" FOREIGN KEY ("toStandardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question" ADD CONSTRAINT "question_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_translation" ADD CONSTRAINT "question_translation_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_tags" ADD CONSTRAINT "question_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_tags" ADD CONSTRAINT "question_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_answer" ADD CONSTRAINT "question_answer_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_answer" ADD CONSTRAINT "question_answer_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "question_answer_translation" ADD CONSTRAINT "question_answer_translation_answerId_fkey" FOREIGN KEY ("answerId") REFERENCES "question_answer"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz" ADD CONSTRAINT "quiz_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz" ADD CONSTRAINT "quiz_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz" ADD CONSTRAINT "quiz_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_translation" ADD CONSTRAINT "quiz_translation_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_attempt" ADD CONSTRAINT "quiz_attempt_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_attempt" ADD CONSTRAINT "quiz_attempt_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question_response" ADD CONSTRAINT "quiz_question_response_quizAttemptId_fkey" FOREIGN KEY ("quizAttemptId") REFERENCES "quiz_attempt"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question_response" ADD CONSTRAINT "quiz_question_response_quizQuestionId_fkey" FOREIGN KEY ("quizQuestionId") REFERENCES "quiz_question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question_response_translation" ADD CONSTRAINT "quiz_question_response_translation_responseId_fkey" FOREIGN KEY ("responseId") REFERENCES "quiz_question_response"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question" ADD CONSTRAINT "quiz_question_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question" ADD CONSTRAINT "quiz_question_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "quiz_question_translation" ADD CONSTRAINT "quiz_question_translation_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "quiz_question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder" ADD CONSTRAINT "reminder_reminderListId_fkey" FOREIGN KEY ("reminderListId") REFERENCES "reminder_list"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "reminder_item" ADD CONSTRAINT "reminder_item_reminderId_fkey" FOREIGN KEY ("reminderId") REFERENCES "reminder"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_noteVersionId_fkey" FOREIGN KEY ("noteVersionId") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_projectVersionId_fkey" FOREIGN KEY ("projectVersionId") REFERENCES "project_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_smartContractVersionId_fkey" FOREIGN KEY ("smartContractVersionId") REFERENCES "smart_contract_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_standardVersionId_fkey" FOREIGN KEY ("standardVersionId") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "report" ADD CONSTRAINT "report_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "role" ADD CONSTRAINT "role_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "role_translation" ADD CONSTRAINT "role_translation_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine" ADD CONSTRAINT "routine_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_apiVersionId_fkey" FOREIGN KEY ("apiVersionId") REFERENCES "api_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_smartContractVersionId_fkey" FOREIGN KEY ("smartContractVersionId") REFERENCES "smart_contract_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

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
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project" ADD CONSTRAINT "run_project_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_step" ADD CONSTRAINT "run_project_step_directoryId_fkey" FOREIGN KEY ("directoryId") REFERENCES "project_version_directory"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_step" ADD CONSTRAINT "run_project_step_runProjectId_fkey" FOREIGN KEY ("runProjectId") REFERENCES "run_project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_schedule" ADD CONSTRAINT "run_project_schedule_id_fkey" FOREIGN KEY ("id") REFERENCES "run_project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_schedule_labels" ADD CONSTRAINT "run_project_schedule_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "run_project_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_schedule_labels" ADD CONSTRAINT "run_project_schedule_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_project_schedule_translation" ADD CONSTRAINT "run_project_schedule_translation_runProjectScheduleId_fkey" FOREIGN KEY ("runProjectScheduleId") REFERENCES "run_project_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_routineVersionId_fkey" FOREIGN KEY ("routineVersionId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_runProjectId_fkey" FOREIGN KEY ("runProjectId") REFERENCES "run_project"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine" ADD CONSTRAINT "run_routine_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_input" ADD CONSTRAINT "run_routine_input_inputId_fkey" FOREIGN KEY ("inputId") REFERENCES "routine_version_input"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_input" ADD CONSTRAINT "run_routine_input_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_nodeId_fkey" FOREIGN KEY ("nodeId") REFERENCES "node"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_runRoutineId_fkey" FOREIGN KEY ("runRoutineId") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_step" ADD CONSTRAINT "run_routine_step_subroutineVersionId_fkey" FOREIGN KEY ("subroutineVersionId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_schedule" ADD CONSTRAINT "run_routine_schedule_id_fkey" FOREIGN KEY ("id") REFERENCES "run_routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_schedule_labels" ADD CONSTRAINT "run_routine_schedule_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "run_routine_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_schedule_labels" ADD CONSTRAINT "run_routine_schedule_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "run_routine_schedule_translation" ADD CONSTRAINT "run_routine_schedule_translation_runRoutineScheduleId_fkey" FOREIGN KEY ("runRoutineScheduleId") REFERENCES "run_routine_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract" ADD CONSTRAINT "smart_contract_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract" ADD CONSTRAINT "smart_contract_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract" ADD CONSTRAINT "smart_contract_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "smart_contract_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract" ADD CONSTRAINT "smart_contract_ownedByUserId_fkey" FOREIGN KEY ("ownedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_version" ADD CONSTRAINT "smart_contract_version_pullRequestId_fkey" FOREIGN KEY ("pullRequestId") REFERENCES "pull_request"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_version" ADD CONSTRAINT "smart_contract_version_rootId_fkey" FOREIGN KEY ("rootId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_version" ADD CONSTRAINT "smart_contract_version_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_version_translation" ADD CONSTRAINT "smart_contract_version_translation_smartContractVersionId_fkey" FOREIGN KEY ("smartContractVersionId") REFERENCES "smart_contract_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_tags" ADD CONSTRAINT "smart_contract_tags_tagTag_fkey" FOREIGN KEY ("tagTag") REFERENCES "tag"("tag") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_tags" ADD CONSTRAINT "smart_contract_tags_taggedId_fkey" FOREIGN KEY ("taggedId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_labels" ADD CONSTRAINT "smart_contract_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "smart_contract_labels" ADD CONSTRAINT "smart_contract_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "standard" ADD CONSTRAINT "standard_ownedByOrganizationId_fkey" FOREIGN KEY ("ownedByOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

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
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_questionAnswerId_fkey" FOREIGN KEY ("questionAnswerId") REFERENCES "question_answer"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "bookmark" ADD CONSTRAINT "bookmark_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_api" ADD CONSTRAINT "stats_api_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_organization" ADD CONSTRAINT "stats_organization_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_project" ADD CONSTRAINT "stats_project_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_quiz" ADD CONSTRAINT "stats_quiz_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_routine" ADD CONSTRAINT "stats_routine_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_smart_contract" ADD CONSTRAINT "stats_smart_contract_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_standard" ADD CONSTRAINT "stats_standard_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "stats_user" ADD CONSTRAINT "stats_user_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag" ADD CONSTRAINT "tag_createdById_fkey" FOREIGN KEY ("createdById") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "tag_translation" ADD CONSTRAINT "tag_translation_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_fromUserId_fkey" FOREIGN KEY ("fromUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_fromOrganizationId_fkey" FOREIGN KEY ("fromOrganizationId") REFERENCES "organization"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_toUserId_fkey" FOREIGN KEY ("toUserId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_toOrganizationId_fkey" FOREIGN KEY ("toOrganizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "transfer" ADD CONSTRAINT "transfer_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user" ADD CONSTRAINT "user_invitedByUserId_fkey" FOREIGN KEY ("invitedByUserId") REFERENCES "user"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user" ADD CONSTRAINT "user_premiumId_fkey" FOREIGN KEY ("premiumId") REFERENCES "premium"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_translation" ADD CONSTRAINT "user_translation_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_language" ADD CONSTRAINT "user_language_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule" ADD CONSTRAINT "user_schedule_reminderListId_fkey" FOREIGN KEY ("reminderListId") REFERENCES "reminder_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule" ADD CONSTRAINT "user_schedule_resourceListId_fkey" FOREIGN KEY ("resourceListId") REFERENCES "resource_list"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule" ADD CONSTRAINT "user_schedule_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule_labels" ADD CONSTRAINT "user_schedule_labels_labelledId_fkey" FOREIGN KEY ("labelledId") REFERENCES "user_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule_labels" ADD CONSTRAINT "user_schedule_labels_labelId_fkey" FOREIGN KEY ("labelId") REFERENCES "label"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule_filter" ADD CONSTRAINT "user_schedule_filter_userScheduleId_fkey" FOREIGN KEY ("userScheduleId") REFERENCES "user_schedule"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_schedule_filter" ADD CONSTRAINT "user_schedule_filter_tagId_fkey" FOREIGN KEY ("tagId") REFERENCES "tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "view" ADD CONSTRAINT "view_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_byId_fkey" FOREIGN KEY ("byId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_apiId_fkey" FOREIGN KEY ("apiId") REFERENCES "api"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_commentId_fkey" FOREIGN KEY ("commentId") REFERENCES "comment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_issueId_fkey" FOREIGN KEY ("issueId") REFERENCES "issue"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_noteId_fkey" FOREIGN KEY ("noteId") REFERENCES "note"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_postId_fkey" FOREIGN KEY ("postId") REFERENCES "post"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_projectId_fkey" FOREIGN KEY ("projectId") REFERENCES "project"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_questionId_fkey" FOREIGN KEY ("questionId") REFERENCES "question"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_questionAnswerId_fkey" FOREIGN KEY ("questionAnswerId") REFERENCES "question_answer"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_quizId_fkey" FOREIGN KEY ("quizId") REFERENCES "quiz"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_routineId_fkey" FOREIGN KEY ("routineId") REFERENCES "routine"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_smartContractId_fkey" FOREIGN KEY ("smartContractId") REFERENCES "smart_contract"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "vote" ADD CONSTRAINT "vote_standardId_fkey" FOREIGN KEY ("standardId") REFERENCES "standard"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_organizationId_fkey" FOREIGN KEY ("organizationId") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "wallet" ADD CONSTRAINT "wallet_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_api_versionToproject_version_directory" ADD CONSTRAINT "_api_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "api_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_api_versionToproject_version_directory" ADD CONSTRAINT "_api_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_note_versionToproject_version_directory" ADD CONSTRAINT "_note_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "note_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_note_versionToproject_version_directory" ADD CONSTRAINT "_note_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_organizationToproject_version_directory" ADD CONSTRAINT "_organizationToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_organizationToproject_version_directory" ADD CONSTRAINT "_organizationToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

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
ALTER TABLE "_project_version_directoryTosmart_contract_version" ADD CONSTRAINT "_project_version_directoryTosmart_contract_version_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryTosmart_contract_version" ADD CONSTRAINT "_project_version_directoryTosmart_contract_version_B_fkey" FOREIGN KEY ("B") REFERENCES "smart_contract_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryTostandard_version" ADD CONSTRAINT "_project_version_directoryTostandard_version_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryTostandard_version" ADD CONSTRAINT "_project_version_directoryTostandard_version_B_fkey" FOREIGN KEY ("B") REFERENCES "standard_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;
