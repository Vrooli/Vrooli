-- AlterTable
ALTER TABLE "user" ADD COLUMN     "isPrivateNotes" BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN     "isPrivateNotesCreated" BOOLEAN NOT NULL DEFAULT false;
