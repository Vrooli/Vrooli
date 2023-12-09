/*
  Warnings:

  - You are about to drop the column `forkId` on the `chat_message` table. All the data in the column will be lost.
  - You are about to drop the column `isFork` on the `chat_message` table. All the data in the column will be lost.

*/
-- DropForeignKey
ALTER TABLE "chat_message" DROP CONSTRAINT "chat_message_forkId_fkey";

-- AlterTable
ALTER TABLE "chat_message" DROP COLUMN "forkId",
DROP COLUMN "isFork",
ADD COLUMN     "parentId" UUID,
ADD COLUMN     "sequence" SERIAL NOT NULL,
ADD COLUMN     "versionIndex" INTEGER NOT NULL DEFAULT 0;

-- AddForeignKey
ALTER TABLE "chat_message" ADD CONSTRAINT "chat_message_parentId_fkey" FOREIGN KEY ("parentId") REFERENCES "chat_message"("id") ON DELETE SET NULL ON UPDATE CASCADE;
