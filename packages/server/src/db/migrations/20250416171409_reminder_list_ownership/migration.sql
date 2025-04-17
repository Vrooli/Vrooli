/*
  Warnings:

  - Added the required column `userId` to the `reminder_list` table without a default value. This is not possible if the table is not empty.

*/

-- Delete all existing reminder_lists
DELETE FROM "reminder_list";

-- AlterTable
ALTER TABLE "reminder_list" ADD COLUMN     "userId" UUID NOT NULL;

-- AddForeignKey
ALTER TABLE "reminder_list" ADD CONSTRAINT "reminder_list_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;
