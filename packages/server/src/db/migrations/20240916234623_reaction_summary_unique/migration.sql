/*
  Warnings:

  - A unique constraint covering the columns `[emoji,apiId,chatMessageId,codeId,commentId,issueId,noteId,postId,projectId,questionId,questionAnswerId,quizId,routineId,standardId]` on the table `reaction_summary` will be added. If there are existing duplicate values, this will fail.

*/
-- CreateIndex
CREATE UNIQUE INDEX "reaction_summary_emoji_apiId_chatMessageId_codeId_commentId_key" ON "reaction_summary"("emoji", "apiId", "chatMessageId", "codeId", "commentId", "issueId", "noteId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "standardId");
