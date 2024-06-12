/*
  Warnings:

  - You are about to drop the `_project_version_directoryTocode_version` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `_teamToproject_version_directory` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "_project_version_directoryTocode_version" DROP CONSTRAINT "_project_version_directoryTosmart_contract_version_A_fkey";

-- DropForeignKey
ALTER TABLE "_project_version_directoryTocode_version" DROP CONSTRAINT "_project_version_directoryTosmart_contract_version_B_fkey";

-- DropForeignKey
ALTER TABLE "_teamToproject_version_directory" DROP CONSTRAINT "_organizationToproject_version_directory_A_fkey";

-- DropForeignKey
ALTER TABLE "_teamToproject_version_directory" DROP CONSTRAINT "_organizationToproject_version_directory_B_fkey";

-- AlterTable
ALTER TABLE "code" RENAME CONSTRAINT "smart_contract_pkey" TO "code_pkey";

-- AlterTable
ALTER TABLE "code_labels" RENAME CONSTRAINT "smart_contract_labels_pkey" TO "code_labels_pkey";

-- AlterTable
ALTER TABLE "code_tags" RENAME CONSTRAINT "smart_contract_tags_pkey" TO "code_tags_pkey";

-- AlterTable
ALTER TABLE "code_version" RENAME CONSTRAINT "smart_contract_version_pkey" TO "code_version_pkey";

-- AlterTable
ALTER TABLE "code_version_translation" RENAME CONSTRAINT "smart_contract_version_translation_pkey" TO "code_version_translation_pkey";

-- AlterTable
ALTER TABLE "stats_code" RENAME CONSTRAINT "stats_smart_contract_pkey" TO "stats_code_pkey";

-- AlterTable
ALTER TABLE "stats_team" RENAME CONSTRAINT "stats_organization_pkey" TO "stats_team_pkey";

-- AlterTable
ALTER TABLE "team" RENAME CONSTRAINT "organization_pkey" TO "team_pkey";

-- AlterTable
ALTER TABLE "team_language" RENAME CONSTRAINT "organization_language_pkey" TO "team_language_pkey";

-- AlterTable
ALTER TABLE "team_tags" RENAME CONSTRAINT "organization_tags_pkey" TO "team_tags_pkey";

-- AlterTable
ALTER TABLE "team_translation" RENAME CONSTRAINT "organization_translation_pkey" TO "team_translation_pkey";

-- DropTable
DROP TABLE "_project_version_directoryTocode_version";

-- DropTable
DROP TABLE "_teamToproject_version_directory";

-- RenameForeignKey
ALTER TABLE "api" RENAME CONSTRAINT "api_ownedByOrganizationId_fkey" TO "api_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "api_key" RENAME CONSTRAINT "api_key_organizationId_fkey" TO "api_key_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "bookmark" RENAME CONSTRAINT "bookmark_organizationId_fkey" TO "bookmark_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "bookmark" RENAME CONSTRAINT "bookmark_smartContractId_fkey" TO "bookmark_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "chat" RENAME CONSTRAINT "chat_organizationId_fkey" TO "chat_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "code" RENAME CONSTRAINT "smart_contract_createdById_fkey" TO "code_createdById_fkey";

-- RenameForeignKey
ALTER TABLE "code" RENAME CONSTRAINT "smart_contract_ownedByOrganizationId_fkey" TO "code_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "code" RENAME CONSTRAINT "smart_contract_ownedByUserId_fkey" TO "code_ownedByUserId_fkey";

-- RenameForeignKey
ALTER TABLE "code" RENAME CONSTRAINT "smart_contract_parentId_fkey" TO "code_parentId_fkey";

-- RenameForeignKey
ALTER TABLE "code_labels" RENAME CONSTRAINT "smart_contract_labels_labelId_fkey" TO "code_labels_labelId_fkey";

-- RenameForeignKey
ALTER TABLE "code_labels" RENAME CONSTRAINT "smart_contract_labels_labelledId_fkey" TO "code_labels_labelledId_fkey";

-- RenameForeignKey
ALTER TABLE "code_tags" RENAME CONSTRAINT "smart_contract_tags_tagTag_fkey" TO "code_tags_tagTag_fkey";

-- RenameForeignKey
ALTER TABLE "code_tags" RENAME CONSTRAINT "smart_contract_tags_taggedId_fkey" TO "code_tags_taggedId_fkey";

-- RenameForeignKey
ALTER TABLE "code_version" RENAME CONSTRAINT "smart_contract_version_pullRequestId_fkey" TO "code_version_pullRequestId_fkey";

-- RenameForeignKey
ALTER TABLE "code_version" RENAME CONSTRAINT "smart_contract_version_resourceListId_fkey" TO "code_version_resourceListId_fkey";

-- RenameForeignKey
ALTER TABLE "code_version" RENAME CONSTRAINT "smart_contract_version_rootId_fkey" TO "code_version_rootId_fkey";

-- RenameForeignKey
ALTER TABLE "code_version_translation" RENAME CONSTRAINT "smart_contract_version_translation_smartContractVersionId_fkey" TO "code_version_translation_codeVersionId_fkey";

-- RenameForeignKey
ALTER TABLE "comment" RENAME CONSTRAINT "comment_ownedByOrganizationId_fkey" TO "comment_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "comment" RENAME CONSTRAINT "comment_smartContractVersionId_fkey" TO "comment_codeVersionId_fkey";

-- RenameForeignKey
ALTER TABLE "email" RENAME CONSTRAINT "email_organizationId_fkey" TO "email_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "issue" RENAME CONSTRAINT "issue_organizationId_fkey" TO "issue_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "issue" RENAME CONSTRAINT "issue_smartContractId_fkey" TO "issue_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "label" RENAME CONSTRAINT "label_ownedByOrganizationId_fkey" TO "label_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "meeting" RENAME CONSTRAINT "meeting_organizationId_fkey" TO "meeting_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "member" RENAME CONSTRAINT "member_organizationId_fkey" TO "member_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "member_invite" RENAME CONSTRAINT "member_invite_organizationId_fkey" TO "member_invite_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "note" RENAME CONSTRAINT "note_ownedByOrganizationId_fkey" TO "note_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "notification_subscription" RENAME CONSTRAINT "notification_subscription_organizationId_fkey" TO "notification_subscription_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "notification_subscription" RENAME CONSTRAINT "notification_subscription_smartContractId_fkey" TO "notification_subscription_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "payment" RENAME CONSTRAINT "payment_organizationId_fkey" TO "payment_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "phone" RENAME CONSTRAINT "phone_organizationId_fkey" TO "phone_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "post" RENAME CONSTRAINT "post_organizationId_fkey" TO "post_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "project" RENAME CONSTRAINT "project_ownedByOrganizationId_fkey" TO "project_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "pull_request" RENAME CONSTRAINT "pull_request_toSmartContractId_fkey" TO "pull_request_toCodeId_fkey";

-- RenameForeignKey
ALTER TABLE "question" RENAME CONSTRAINT "question_organizationId_fkey" TO "question_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "question" RENAME CONSTRAINT "question_smartContractId_fkey" TO "question_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "reaction" RENAME CONSTRAINT "reaction_smartContractId_fkey" TO "reaction_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "reaction_summary" RENAME CONSTRAINT "reaction_summary_smartContractId_fkey" TO "reaction_summary_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "report" RENAME CONSTRAINT "report_organizationId_fkey" TO "report_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "report" RENAME CONSTRAINT "report_smartContractVersionId_fkey" TO "report_codeVersionId_fkey";

-- RenameForeignKey
ALTER TABLE "role" RENAME CONSTRAINT "role_organizationId_fkey" TO "role_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "routine" RENAME CONSTRAINT "routine_ownedByOrganizationId_fkey" TO "routine_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "routine_version" RENAME CONSTRAINT "routine_version_smartContractVersionId_fkey" TO "routine_version_codeVersionId_fkey";

-- RenameForeignKey
ALTER TABLE "run_project" RENAME CONSTRAINT "run_project_organizationId_fkey" TO "run_project_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "run_routine" RENAME CONSTRAINT "run_routine_organizationId_fkey" TO "run_routine_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "standard" RENAME CONSTRAINT "standard_ownedByOrganizationId_fkey" TO "standard_ownedByTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "stats_code" RENAME CONSTRAINT "stats_smart_contract_smartContractId_fkey" TO "stats_code_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "stats_team" RENAME CONSTRAINT "stats_organization_organizationId_fkey" TO "stats_team_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "team" RENAME CONSTRAINT "organization_createdById_fkey" TO "team_createdById_fkey";

-- RenameForeignKey
ALTER TABLE "team" RENAME CONSTRAINT "organization_parentId_fkey" TO "team_parentId_fkey";

-- RenameForeignKey
ALTER TABLE "team" RENAME CONSTRAINT "organization_premiumId_fkey" TO "team_premiumId_fkey";

-- RenameForeignKey
ALTER TABLE "team" RENAME CONSTRAINT "organization_resourceListId_fkey" TO "team_resourceListId_fkey";

-- RenameForeignKey
ALTER TABLE "team_language" RENAME CONSTRAINT "organization_language_organizationId_fkey" TO "team_language_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "team_tags" RENAME CONSTRAINT "organization_tags_tagTag_fkey" TO "team_tags_tagTag_fkey";

-- RenameForeignKey
ALTER TABLE "team_tags" RENAME CONSTRAINT "organization_tags_taggedId_fkey" TO "team_tags_taggedId_fkey";

-- RenameForeignKey
ALTER TABLE "team_translation" RENAME CONSTRAINT "organization_translation_organizationId_fkey" TO "team_translation_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "transfer" RENAME CONSTRAINT "transfer_fromOrganizationId_fkey" TO "transfer_fromTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "transfer" RENAME CONSTRAINT "transfer_smartContractId_fkey" TO "transfer_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "transfer" RENAME CONSTRAINT "transfer_toOrganizationId_fkey" TO "transfer_toTeamId_fkey";

-- RenameForeignKey
ALTER TABLE "view" RENAME CONSTRAINT "view_organizationId_fkey" TO "view_teamId_fkey";

-- RenameForeignKey
ALTER TABLE "view" RENAME CONSTRAINT "view_smartContractId_fkey" TO "view_codeId_fkey";

-- RenameForeignKey
ALTER TABLE "wallet" RENAME CONSTRAINT "wallet_organizationId_fkey" TO "wallet_teamId_fkey";

-- AddForeignKey
ALTER TABLE "_project_version_directoryToteam" ADD CONSTRAINT "_project_version_directoryToteam_A_fkey" FOREIGN KEY ("A") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_project_version_directoryToteam" ADD CONSTRAINT "_project_version_directoryToteam_B_fkey" FOREIGN KEY ("B") REFERENCES "team"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_code_versionToproject_version_directory" ADD CONSTRAINT "_code_versionToproject_version_directory_A_fkey" FOREIGN KEY ("A") REFERENCES "code_version"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "_code_versionToproject_version_directory" ADD CONSTRAINT "_code_versionToproject_version_directory_B_fkey" FOREIGN KEY ("B") REFERENCES "project_version_directory"("id") ON DELETE CASCADE ON UPDATE CASCADE;
