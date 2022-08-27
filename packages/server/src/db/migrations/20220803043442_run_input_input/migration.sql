/*
  Warnings:

  - A unique constraint covering the columns `[roleId,language]` on the table `role_translation` will be added. If there are existing duplicate values, this will fail.
  - Added the required column `title` to the `role_translation` table without a default value. This is not possible if the table is not empty.
  - Added the required column `inputId` to the `run_input` table without a default value. This is not possible if the table is not empty.

*/
-- DropForeignKey
ALTER TABLE "user_roles" DROP CONSTRAINT "user_roles_roleId_fkey";

-- DropForeignKey
ALTER TABLE "user_roles" DROP CONSTRAINT "user_roles_userId_fkey";

-- DropIndex
DROP INDEX "role_translation_language_roleId_key";

-- AlterTable
ALTER TABLE "role_translation" ADD COLUMN     "title" VARCHAR(128) NOT NULL,
ALTER COLUMN "description" DROP NOT NULL;

-- AlterTable
ALTER TABLE "run_input" ADD COLUMN     "inputId" UUID NOT NULL;

-- CreateIndex
CREATE UNIQUE INDEX "role_translation_roleId_language_key" ON "role_translation"("roleId", "language");

-- AddForeignKey
ALTER TABLE "run_input" ADD CONSTRAINT "run_input_inputId_fkey" FOREIGN KEY ("inputId") REFERENCES "routine_input"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_roles" ADD CONSTRAINT "user_roles_roleId_fkey" FOREIGN KEY ("roleId") REFERENCES "role"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "user_roles" ADD CONSTRAINT "user_roles_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
