-- This migration renames all instances of "organization" to "team" and "smart_contract" to "code"

-- -- Change enums TODO add another migration for this. It's being fussy
-- BEGIN;
-- -- Step 1: Create an intermediary enum type with both old and new values
-- CREATE TYPE "AwardCategory_intermediary" AS ENUM (
--     'AccountAnniversary', 'AccountNew', 'ApiCreate', 'CodeCreate', 'CommentCreate', 'IssueCreate', 
--     'NoteCreate', 'ObjectBookmark', 'ObjectReact', 'PostCreate', 'ProjectCreate', 'PullRequestCreate', 
--     'PullRequestComplete', 'QuestionAnswer', 'QuestionCreate', 'QuizPass', 'ReportEnd', 'ReportContribute', 
--     'Reputation', 'RunRoutine', 'RunProject', 'RoutineCreate', 'StandardCreate', 'Streak', 
--     'TeamCreate', 'TeamJoin', 'UserInvite',
--     'SmartContractCreate', 'OrganizationCreate', 'OrganizationJoin'  -- Include old values temporarily
-- );
-- -- Step 2: Change the column type to the intermediary enum
-- ALTER TABLE "award" ALTER COLUMN "category"
-- TYPE "AwardCategory_intermediary" USING ("category"::text::"AwardCategory_intermediary");
-- -- Step 3: Update the records to new enum values
-- UPDATE "award"
-- SET "category" = CASE "category"
--     WHEN 'SmartContractCreate' THEN 'CodeCreate'
--     WHEN 'OrganizationCreate' THEN 'TeamCreate'
--     WHEN 'OrganizationJoin' THEN 'TeamJoin'
--     ELSE "category"
-- END;
-- -- Step 4: Create the final enum type without the old values
-- CREATE TYPE "AwardCategory_new" AS ENUM (
--     'AccountAnniversary', 'AccountNew', 'ApiCreate', 'CodeCreate', 'CommentCreate', 'IssueCreate', 
--     'NoteCreate', 'ObjectBookmark', 'ObjectReact', 'PostCreate', 'ProjectCreate', 'PullRequestCreate', 
--     'PullRequestComplete', 'QuestionAnswer', 'QuestionCreate', 'QuizPass', 'ReportEnd', 'ReportContribute', 
--     'Reputation', 'RunRoutine', 'RunProject', 'RoutineCreate', 'StandardCreate', 'Streak', 
--     'TeamCreate', 'TeamJoin', 'UserInvite'
-- );
-- -- Step 5: Change the column type to the final enum
-- ALTER TABLE "award" ALTER COLUMN "category"
-- TYPE "AwardCategory_new" USING ("category"::text::"AwardCategory_new");
-- -- Step 6: Drop the intermediary enum
-- DROP TYPE "AwardCategory_intermediary";
-- COMMIT;

-- Drop old indexes
DROP INDEX IF EXISTS "label_label_ownedByUserId_ownedByOrganizationId_key";
DROP INDEX IF EXISTS "member_organizationId_userId_key";
DROP INDEX IF EXISTS "member_invite_userId_organizationId_key";
DROP INDEX IF EXISTS "role_organizationId_name_key";
DROP INDEX IF EXISTS "organization_handle_key";
DROP INDEX IF EXISTS "organization_premiumId_key";
DROP INDEX IF EXISTS "organization_resourceListId_key";
DROP INDEX IF EXISTS "organization_stripeCustomerId_key";
DROP INDEX IF EXISTS "organization_language_organizationId_language_key";
DROP INDEX IF EXISTS "organization_translation_organizationId_language_key";
DROP INDEX IF EXISTS "organization_tags_taggedId_tagTag_key";
DROP INDEX IF EXISTS "smart_contract_version_resourceListId_key";
DROP INDEX IF EXISTS "smart_contract_version_pullRequestId_key";
DROP INDEX IF EXISTS "smart_contract_version_rootId_versionIndex_key";
DROP INDEX IF EXISTS "smart_contract_version_translation_smartContractVersionId_l_key";
DROP INDEX IF EXISTS "smart_contract_version_translation_smartContractVersionId_language_key";
DROP INDEX IF EXISTS "smart_contract_tags_taggedId_tagTag_key";
DROP INDEX IF EXISTS "smart_contract_labels_labelledId_labelId_key";
DROP INDEX IF EXISTS "_project_version_directoryToorganization_AB_unique";
DROP INDEX IF EXISTS "_project_version_directoryToorganization_B_index";
DROP INDEX IF EXISTS "_smart_contract_versionToproject_version_directory_AB_unique";
DROP INDEX IF EXISTS "_smart_contract_versionToproject_version_directory_B_index";

-- Rename changed columns (where the table name stayed the same)
ALTER TABLE "api" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "api_key" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "bookmark" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "bookmark" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "chat" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "comment" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "comment" RENAME COLUMN "smartContractVersionId" TO "codeVersionId";
ALTER TABLE "email" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "issue" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "issue" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "label" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "meeting" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "meeting" RENAME COLUMN "showOnOrganizationProfile" TO "showOnTeamProfile";
ALTER TABLE "member" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "member_invite" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "note" RENAME COLUMN "createdByOrganizationId" TO "createdByTeamId";
ALTER TABLE "note" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "notification_subscription" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "notification_subscription" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "payment" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "phone" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "post" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "project" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "pull_request" RENAME COLUMN "toSmartContractId" TO "toCodeId";
ALTER TABLE "question" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "question" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "reaction" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "reaction_summary" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "report" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "report" RENAME COLUMN "smartContractVersionId" TO "codeVersionId";
ALTER TABLE "role" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "routine" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "routine_version" RENAME COLUMN "smartContractCallData" TO "codeCallData";
ALTER TABLE "routine_version" RENAME COLUMN "smartContractVersionId" TO "codeVersionId";
ALTER TABLE "run_project" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "run_routine" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "standard" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";
ALTER TABLE "stats_project" RENAME COLUMN "organizations" TO "teams";
ALTER TABLE "stats_project" RENAME COLUMN "smartContracts" TO "codes";
ALTER TABLE "stats_site" RENAME COLUMN "organizationsCreated" TO "teamsCreated";
ALTER TABLE "stats_site" RENAME COLUMN "smartContractCalls" TO "codeCalls";
ALTER TABLE "stats_site" RENAME COLUMN "smartContractCompletionTimeAverage" TO "codeCompletionTimeAverage";
ALTER TABLE "stats_site" RENAME COLUMN "smartContractsCompleted" TO "codesCompleted";
ALTER TABLE "stats_site" RENAME COLUMN "smartContractsCreated" TO "codesCreated";
ALTER TABLE "stats_user" RENAME COLUMN "organizationsCreated" TO "teamsCreated";
ALTER TABLE "stats_user" RENAME COLUMN "smartContractCompletionTimeAverage" TO "codeCompletionTimeAverage";
ALTER TABLE "stats_user" RENAME COLUMN "smartContractsCompleted" TO "codesCompleted";
ALTER TABLE "stats_user" RENAME COLUMN "smartContractsCreated" TO "codesCreated";
ALTER TABLE "transfer" RENAME COLUMN "fromOrganizationId" TO "fromTeamId";
ALTER TABLE "transfer" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "transfer" RENAME COLUMN "toOrganizationId" TO "toTeamId";
ALTER TABLE "user" RENAME COLUMN "isPrivateOrganizationsCreated" TO "isPrivateTeamsCreated";
ALTER TABLE "user" RENAME COLUMN "isPrivateSmartContracts" TO "isPrivateCodes";
ALTER TABLE "user" RENAME COLUMN "isPrivateSmartContractsCreated" TO "isPrivateCodesCreated";
ALTER TABLE "view" RENAME COLUMN "organizationId" TO "teamId";
ALTER TABLE "view" RENAME COLUMN "smartContractId" TO "codeId";
ALTER TABLE "wallet" RENAME COLUMN "organizationId" TO "teamId";

-- Rename internal tables
ALTER TABLE "_organizationToproject_version_directory" RENAME TO "_teamToproject_version_directory";
ALTER TABLE "_project_version_directoryTosmart_contract_version" RENAME TO "_project_version_directoryTocode_version";

-- Create internal tables that didn't exist before
CREATE TABLE "_project_version_directoryToteam" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);
CREATE TABLE "_code_versionToproject_version_directory" (
    "A" UUID NOT NULL,
    "B" UUID NOT NULL
);

-- Rename changed tables (and any columns they have that changed)
ALTER TABLE "organization" RENAME TO "team";

ALTER TABLE "organization_language" RENAME TO "team_language";
ALTER TABLE "team_language" RENAME COLUMN "organizationId" TO "teamId";

ALTER TABLE "organization_tags" RENAME TO "team_tags";

ALTER TABLE "organization_translation" RENAME TO "team_translation";
ALTER TABLE "team_translation" RENAME COLUMN "organizationId" TO "teamId";

ALTER TABLE "smart_contract" RENAME TO "code";
ALTER TABLE "code" RENAME COLUMN "ownedByOrganizationId" TO "ownedByTeamId";

ALTER TABLE "smart_contract_labels" RENAME TO "code_labels";

ALTER TABLE "smart_contract_tags" RENAME TO "code_tags";

ALTER TABLE "smart_contract_version" RENAME TO "code_version";

ALTER TABLE "smart_contract_version_translation" RENAME TO "code_version_translation";
ALTER TABLE "code_version_translation" RENAME COLUMN "smartContractVersionId" TO "codeVersionId";

ALTER TABLE "stats_organization" RENAME TO "stats_team";
ALTER TABLE "stats_team" RENAME COLUMN "smartContracts" TO "codes";
ALTER TABLE "stats_team" RENAME COLUMN "organizationId" TO "teamId";

ALTER TABLE "stats_smart_contract" RENAME TO "stats_code";
ALTER TABLE "stats_code" RENAME COLUMN "smartContractId" TO "codeId";

-- Create indexes
CREATE UNIQUE INDEX "team_handle_key" ON "team"("handle");
CREATE UNIQUE INDEX "team_premiumId_key" ON "team"("premiumId");
CREATE UNIQUE INDEX "team_resourceListId_key" ON "team"("resourceListId");
CREATE UNIQUE INDEX "team_stripeCustomerId_key" ON "team"("stripeCustomerId");
CREATE UNIQUE INDEX "team_language_teamId_language_key" ON "team_language"("teamId", "language");
CREATE UNIQUE INDEX "team_translation_teamId_language_key" ON "team_translation"("teamId", "language");
CREATE UNIQUE INDEX "team_tags_taggedId_tagTag_key" ON "team_tags"("taggedId", "tagTag");
CREATE UNIQUE INDEX "code_version_resourceListId_key" ON "code_version"("resourceListId");
CREATE UNIQUE INDEX "code_version_pullRequestId_key" ON "code_version"("pullRequestId");
CREATE UNIQUE INDEX "code_version_rootId_versionIndex_key" ON "code_version"("rootId", "versionIndex");
CREATE UNIQUE INDEX "code_version_translation_codeVersionId_language_key" ON "code_version_translation"("codeVersionId", "language");
CREATE UNIQUE INDEX "code_tags_taggedId_tagTag_key" ON "code_tags"("taggedId", "tagTag");
CREATE UNIQUE INDEX "code_labels_labelledId_labelId_key" ON "code_labels"("labelledId", "labelId");
CREATE UNIQUE INDEX "_project_version_directoryToteam_AB_unique" ON "_project_version_directoryToteam"("A", "B");
CREATE INDEX "_project_version_directoryToteam_B_index" ON "_project_version_directoryToteam"("B");
CREATE UNIQUE INDEX "_code_versionToproject_version_directory_AB_unique" ON "_code_versionToproject_version_directory"("A", "B");
CREATE INDEX "_code_versionToproject_version_directory_B_index" ON "_code_versionToproject_version_directory"("B");
CREATE UNIQUE INDEX "label_label_ownedByUserId_ownedByTeamId_key" ON "label"("label", "ownedByUserId", "ownedByTeamId");
CREATE UNIQUE INDEX "member_teamId_userId_key" ON "member"("teamId", "userId");
CREATE UNIQUE INDEX "member_invite_userId_teamId_key" ON "member_invite"("userId", "teamId");
CREATE UNIQUE INDEX "role_teamId_name_key" ON "role"("teamId", "name");