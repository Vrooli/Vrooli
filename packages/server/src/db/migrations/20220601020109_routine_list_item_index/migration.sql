/*
  Warnings:

  - Made the column `index` on table `node_routine_list_item` required. This step will fail if there are existing NULL values in that column.

*/
-- AlterTable
ALTER TABLE "node_routine_list_item" ALTER COLUMN "index" SET NOT NULL,
ALTER COLUMN "index" DROP DEFAULT;
