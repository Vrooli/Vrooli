-- CreateEnum
CREATE TYPE "FocusModeStopCondition" AS ENUM ('AfterStopTime', 'Automatic', 'Never', 'NextBegins');

-- CreateTable
CREATE TABLE "active_focus_mode" (
    "id" UUID NOT NULL,
    "userId" UUID NOT NULL,
    "focusModeId" UUID NOT NULL,
    "stopCondition" "FocusModeStopCondition" NOT NULL DEFAULT 'Automatic',
    "stopTime" TIMESTAMPTZ(6),
    "created_at" TIMESTAMPTZ(6) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMPTZ(6) NOT NULL,

    CONSTRAINT "active_focus_mode_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "active_focus_mode_userId_key" ON "active_focus_mode"("userId");

-- AddForeignKey
ALTER TABLE "active_focus_mode" ADD CONSTRAINT "active_focus_mode_userId_fkey" FOREIGN KEY ("userId") REFERENCES "user"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "active_focus_mode" ADD CONSTRAINT "active_focus_mode_focusModeId_fkey" FOREIGN KEY ("focusModeId") REFERENCES "focus_mode"("id") ON DELETE CASCADE ON UPDATE CASCADE;
