/*
  Warnings:

  - A unique constraint covering the columns `[userId,focusModeId]` on the table `active_focus_mode` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateIndex
CREATE UNIQUE INDEX "active_focus_mode_userId_focusModeId_key" ON "active_focus_mode"("userId", "focusModeId");
