/*
  Warnings:

  - You are about to drop the column `configCallData` on the `routine_version` table. All the data in the column will be lost.
  - You are about to drop the column `configFormInput` on the `routine_version` table. All the data in the column will be lost.
  - You are about to drop the column `configFormOutput` on the `routine_version` table. All the data in the column will be lost.
  - You are about to drop the `node` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_end` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_end_next` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_link` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_link_when` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_link_when_translation` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_loop` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_loop_while` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_loop_while_translation` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_routine_list` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_routine_list_item` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_routine_list_item_translation` table. If the table is not empty, all the data it contains will be lost.
  - You are about to drop the `node_translation` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "node" DROP CONSTRAINT "node_routineVersionId_fkey";

-- DropForeignKey
ALTER TABLE "node_end" DROP CONSTRAINT "node_end_nodeId_fkey";

-- DropForeignKey
ALTER TABLE "node_end_next" DROP CONSTRAINT "node_end_next_fromEndId_fkey";

-- DropForeignKey
ALTER TABLE "node_end_next" DROP CONSTRAINT "node_end_next_toRoutineVersionId_fkey";

-- DropForeignKey
ALTER TABLE "node_link" DROP CONSTRAINT "node_link_fromId_fkey";

-- DropForeignKey
ALTER TABLE "node_link" DROP CONSTRAINT "node_link_routineVersionId_fkey";

-- DropForeignKey
ALTER TABLE "node_link" DROP CONSTRAINT "node_link_toId_fkey";

-- DropForeignKey
ALTER TABLE "node_link_when" DROP CONSTRAINT "node_link_when_linkId_fkey";

-- DropForeignKey
ALTER TABLE "node_link_when_translation" DROP CONSTRAINT "node_link_when_translation_whenId_fkey";

-- DropForeignKey
ALTER TABLE "node_loop" DROP CONSTRAINT "node_loop_nodeId_fkey";

-- DropForeignKey
ALTER TABLE "node_loop_while" DROP CONSTRAINT "node_loop_while_loopId_fkey";

-- DropForeignKey
ALTER TABLE "node_loop_while_translation" DROP CONSTRAINT "node_loop_while_translation_whileId_fkey";

-- DropForeignKey
ALTER TABLE "node_routine_list" DROP CONSTRAINT "node_routine_list_nodeId_fkey";

-- DropForeignKey
ALTER TABLE "node_routine_list_item" DROP CONSTRAINT "node_routine_list_item_listId_fkey";

-- DropForeignKey
ALTER TABLE "node_routine_list_item" DROP CONSTRAINT "node_routine_list_item_routineVersionId_fkey";

-- DropForeignKey
ALTER TABLE "node_routine_list_item_translation" DROP CONSTRAINT "node_routine_list_item_translation_itemId_fkey";

-- DropForeignKey
ALTER TABLE "node_translation" DROP CONSTRAINT "node_translation_nodeId_fkey";

-- DropForeignKey
ALTER TABLE "run_routine_step" DROP CONSTRAINT "run_routine_step_nodeId_fkey";

-- AlterTable
ALTER TABLE "_api_versionToproject_version_directory" ADD CONSTRAINT "_api_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_api_versionToproject_version_directory_AB_unique";

-- AlterTable
ALTER TABLE "_code_versionToproject_version_directory" ADD CONSTRAINT "_code_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_code_versionToproject_version_directory_AB_unique";

-- AlterTable
ALTER TABLE "_memberTorole" ADD CONSTRAINT "_memberTorole_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_memberTorole_AB_unique";

-- AlterTable
ALTER TABLE "_note_versionToproject_version_directory" ADD CONSTRAINT "_note_versionToproject_version_directory_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_note_versionToproject_version_directory_AB_unique";

-- AlterTable
ALTER TABLE "_project_version_directoryToroutine_version" ADD CONSTRAINT "_project_version_directoryToroutine_version_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_project_version_directoryToroutine_version_AB_unique";

-- AlterTable
ALTER TABLE "_project_version_directoryTostandard_version" ADD CONSTRAINT "_project_version_directoryTostandard_version_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_project_version_directoryTostandard_version_AB_unique";

-- AlterTable
ALTER TABLE "_project_version_directoryToteam" ADD CONSTRAINT "_project_version_directoryToteam_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_project_version_directoryToteam_AB_unique";

-- AlterTable
ALTER TABLE "_project_version_directory_listing" ADD CONSTRAINT "_project_version_directory_listing_AB_pkey" PRIMARY KEY ("A", "B");

-- DropIndex
DROP INDEX "_project_version_directory_listing_AB_unique";

-- AlterTable
ALTER TABLE "routine_version" DROP COLUMN "configCallData",
DROP COLUMN "configFormInput",
DROP COLUMN "configFormOutput",
ADD COLUMN     "config" VARCHAR(32768),
ADD COLUMN     "multiStepParentId" UUID;

-- DropTable
DROP TABLE "node";

-- DropTable
DROP TABLE "node_end";

-- DropTable
DROP TABLE "node_end_next";

-- DropTable
DROP TABLE "node_link";

-- DropTable
DROP TABLE "node_link_when";

-- DropTable
DROP TABLE "node_link_when_translation";

-- DropTable
DROP TABLE "node_loop";

-- DropTable
DROP TABLE "node_loop_while";

-- DropTable
DROP TABLE "node_loop_while_translation";

-- DropTable
DROP TABLE "node_routine_list";

-- DropTable
DROP TABLE "node_routine_list_item";

-- DropTable
DROP TABLE "node_routine_list_item_translation";

-- DropTable
DROP TABLE "node_translation";

-- AddForeignKey
ALTER TABLE "routine_version" ADD CONSTRAINT "routine_version_multiStepParentId_fkey" FOREIGN KEY ("multiStepParentId") REFERENCES "routine_version"("id") ON DELETE SET NULL ON UPDATE CASCADE;
