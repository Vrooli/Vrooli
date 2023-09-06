/*
  Warnings:

  - The `handle` column on the `organization` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `handle` column on the `project` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - The `handle` column on the `user` table would be dropped and recreated. This will lead to data loss if there is data in the column.
  - You are about to drop the `handle` table. If the table is not empty, all the data it contains will be lost.

*/
-- DropForeignKey
ALTER TABLE "handle" DROP CONSTRAINT "handle_walletId_fkey";

-- AlterTable
ALTER TABLE "organization" DROP COLUMN "handle",
ADD COLUMN     "handle" CITEXT;

-- AlterTable
ALTER TABLE "project" DROP COLUMN "handle",
ADD COLUMN     "handle" CITEXT;

-- AlterTable
ALTER TABLE "user" DROP COLUMN "handle",
ADD COLUMN     "handle" CITEXT;

-- DropTable
DROP TABLE "handle";

-- CreateIndex
CREATE UNIQUE INDEX "organization_handle_key" ON "organization"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "project_handle_key" ON "project"("handle");

-- CreateIndex
CREATE UNIQUE INDEX "user_handle_key" ON "user"("handle");
