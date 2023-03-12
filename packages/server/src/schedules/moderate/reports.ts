import { ReportSuggestedAction } from "@shared/consts";
import pkg, { ReportStatus } from '@prisma/client';
import { PrismaType } from "../../types";
const { PrismaClient } = pkg;

// Constants for calculating when a moderation action for a report should be accepted
// Minimum reputation sum of users who have suggested a specific moderation action
const MIN_REP_FOR_DELETE = 500;
const MIN_REP_FOR_HIDE = 100;
const MIN_REP_FOR_NON_ISSUE = 100;
const MIN_REP_FOR_FALSE_REPORT = 100;
const MIN_REP_FOR_SUSPEND = 2500;
// If report does not meet the minimum reputation for any of the above actions, this is the timeout before 
// the highest voted action is accepted
const DEFAULT_TIMEOUT = 1000 * 60 * 60 * 24 * 7; // 1 week

/**
 * Checks if a report should be closed, with its suggested actions executed.
 * If so, performs the moderation action and triggers appropriate event.
 */
const moderateReport = async (
    report: pkg.report & { 
        responses: (pkg.report_response & { createdBy: pkg.user })[] 
    },
    prisma: PrismaType,
): Promise<void> => {
    let acceptedAction: ReportSuggestedAction | null = null;
    // Group responses by action and sum reputation
    const sums = report?.responses
        ?.reduce((acc, cur) => {
            const action = cur.actionSuggested;
            const rep = cur.createdBy.reputation;
            if (acc[action]) {
                acc[action] += rep;
            } else {
                acc[action] = rep;
            }
            return acc;
        }, {} as Record<ReportSuggestedAction, number>) ?? {};
    // Find the action with the highest reputation sum
    asdfasdf
    // If this action is gte the minimum reputation, accept it
    asdfasdfasdf
    // Otherwise, if the report is old enough, accept the highest voted action 
    if (!acceptedAction && report.created_at < (Date.now() - DEFAULT_TIMEOUT)) {
        // If there is a highest voted action, accept it
        safdasfdasfd
        // If there is a tie, pick the least severe action
        fsdfasdf
        // If there are no responses, mark as non-issue
        asdfasdfsdf
    }
    // If accepted
    if (acceptedAction) {
        // Update report
        asdfasdfsafd
        // Perform moderation action
        asdfasdfs
        // Trigger event
        asdfasdf
    }
}

/**
 * Handles the moderation of content through report suggestions. It works like this: 
 * 1. A user reports some object with a reason and optional details. The reporter can delete their own report at any time.
 * 2. Other users (cannot be report creator or object owner) suggest a moderation action. These include:
 *      - Deleting the object
 *      - Marking the report as a false report
 *      - Hiding the object until a new version is published that fixes the issue
 *      - Marking the report as a non-issue
 *      - Suspending the object owner (for how long depends on previous suspensions)
 * 3. When enough suggestions are made (accounting for the reputation of the report creator and suggestors), 
 *   the report is automatically accepted and the object is moderated accordingly.
 * 4. Notifications are sent to the relevant users when a decision is made, and reputation scores are updated.
 */
export const checkReportResponses = async () => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // We may be dealing with a lot of data, so we need to do this in batches
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all open reports
        const batch = await prisma.report.findMany({
            where: {
                status: ReportStatus.Open,
            },
            select: {
                id: true,
                created_at: true,
                apiVersionId: true,
                commentId: true,
                issueId: true,
                noteVersionId: true,
                organizationId: true,
                postId: true,
                projectVersionId: true,
                questionId: true,
                routineVersionId: true,
                smartContractVersionId: true,
                standardVersionId: true,
                tagId: true,
                userId: true,
                responses: {
                    select: {
                        id: true,
                        actionSuggested: true,
                        createdBy: {
                            select: { reputation: true }
                        }
                    }
                }
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each report, call moderateReport
        for (const report of batch) {
            await moderateReport(report as any, prisma);
        }
    } while (currentBatchSize === batchSize);
    // Close the Prisma client
    await prisma.$disconnect();
}