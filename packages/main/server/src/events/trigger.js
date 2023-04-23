import { setupVerificationCode } from "../auth";
import { isObjectSubscribable, Notify } from "../notify";
import { Award, objectAwardCategory } from "./awards";
import { objectReputationEvent, Reputation } from "./reputation";
export const Trigger = (prisma, languages) => ({
    acountNew: async (userId, emailAddress) => {
        if (emailAddress)
            await setupVerificationCode(emailAddress, prisma, languages);
        Award(prisma, userId, languages).update("AccountNew", 1);
    },
    issueActivity: async ({ issueCreatedById, issueHasBeenClosedOrRejected, issueId, issueStatus, objectId, objectOwner, objectType, userUpdatingIssueId, }) => {
        if (!issueHasBeenClosedOrRejected) {
            if (issueStatus === "Open") {
                await Award(prisma, issueCreatedById, languages).update("IssueCreate", 1);
            }
            else if (issueStatus === "ClosedResolved") {
                await Reputation().update("IssueCreatedWasAccepted", prisma, issueCreatedById);
            }
            else if (issueStatus === "Rejected") {
                await Reputation().update("IssueCreatedWasRejected", prisma, issueCreatedById);
            }
        }
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushIssueStatusChange(issueId, objectId, objectType, issueStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingIssueId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingIssueId);
            }
            notification.toSubscribers(objectType, objectId, userUpdatingIssueId);
        }
    },
    objectCreated: async ({ createdById, hasCompleteAndPublic, hasParent, owner, objectId, objectType, projectId, }) => {
        const awardCategory = objectAwardCategory(objectType);
        if (!hasParent && awardCategory) {
            await Award(prisma, createdById, languages).update(awardCategory, 1);
        }
        const reputationEvent = objectReputationEvent(objectType);
        if (!hasParent && reputationEvent && hasCompleteAndPublic) {
            await Reputation().update(reputationEvent, prisma, createdById, objectId);
        }
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable && owner.__typename === "Organization") {
            const notification = Notify(prisma, languages).pushNewObjectInOrganization(objectType, objectId, owner.id);
            notification.toOrganization(owner.id, createdById);
            notification.toSubscribers("Organization", owner.id, createdById);
        }
        if (isSubscribable && projectId) {
            const notification = Notify(prisma, languages).pushNewObjectInProject(objectType, objectId, projectId);
            notification.toOwner(owner, createdById);
            notification.toSubscribers("Project", projectId, createdById);
        }
        if (["Email", "Phone", "Wallet"].includes(objectType)) {
            Notify(prisma, languages).pushNewDeviceSignIn().toUser(createdById);
        }
    },
    objectUpdated: async ({ updatedById, hasCompleteAndPublic, hasParent, owner, objectId, objectType, originalProjectId, projectId, wasCompleteAndPublic, }) => {
        if (!hasParent && !wasCompleteAndPublic && hasCompleteAndPublic) {
            const reputationEvent = objectReputationEvent(objectType);
            if (reputationEvent) {
                await Reputation().update(reputationEvent, prisma, updatedById, objectId);
            }
        }
        if (projectId && projectId !== originalProjectId) {
            const isSubscribable = isObjectSubscribable(objectType);
            if (isSubscribable) {
                const notification = Notify(prisma, languages).pushNewObjectInProject(objectType, objectId, projectId);
                notification.toOwner(owner, updatedById);
                notification.toSubscribers("Project", projectId, updatedById);
            }
        }
    },
    objectDeleted: async ({ deletedById, hasBeenTransferred, hasParent, objectId, objectType, wasCompleteAndPublic, }) => {
        if (!hasParent && !hasBeenTransferred && wasCompleteAndPublic) {
            const reputationEvent = objectReputationEvent(objectType);
            if (reputationEvent) {
                await Reputation().unCreateObject(prisma, objectId, deletedById);
            }
        }
    },
    objectCopy: async (owner, forkedByUserId, objectType, parentId) => {
    },
    objectBookmark: async (isBookmarked, objectType, objectId, userId) => {
    },
    objectReact: async (previousReaction, currentReaction, objectType, objectId, userId) => {
    },
    organizationJoin: async (organizationId, userId) => {
    },
    pullRequestActivity: async ({ pullRequestCreatedById, pullRequestHasBeenClosedOrRejected, pullRequestId, pullRequestStatus, objectId, objectOwner, objectType, userUpdatingPullRequestId, }) => {
        if (!pullRequestHasBeenClosedOrRejected) {
            if (pullRequestStatus === "Open") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestCreate", 1);
            }
            else if (pullRequestStatus === "Canceled") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestCreate", -1);
            }
            else if (pullRequestStatus === "Merged") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestComplete", 1);
                await Reputation().update("PullRequestWasAccepted", prisma, pullRequestCreatedById);
            }
            else if (pullRequestStatus === "Rejected") {
                await Reputation().update("PullRequestWasRejected", prisma, pullRequestCreatedById);
            }
        }
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushPullRequestStatusChange(pullRequestId, objectId, objectType, pullRequestStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingPullRequestId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingPullRequestId);
            }
            notification.toSubscribers(objectType, objectId, userUpdatingPullRequestId);
        }
    },
    questionAccepted: async (questionId, answerId, userId) => {
    },
    questionAnswer: async (questionId, userId) => {
    },
    reportActivity: async ({ objectId, objectOwner, objectType, reportContributors, reportCreatedById, reportId, reportStatus, userUpdatingReportId, }) => {
        if (reportStatus !== "Open") {
            const contributors = [...new Set(reportContributors)];
            for (const contributorId of contributors) {
                const award = await Award(prisma, contributorId, languages).update("ReportContribute", 1);
                await Reputation().updateReportContribute(award.progress, prisma, contributorId);
            }
        }
        if (reportCreatedById && ["ClosedDeleted", "ClosedResolved", "ClosedSuspended"].includes(reportStatus)) {
            await Award(prisma, reportCreatedById, languages).update("ReportEnd", 1);
            await Reputation().update("ReportWasAccepted", prisma, reportCreatedById);
        }
        if (reportCreatedById && ["ClosedFalseReport", "ClosedNonIssue"].includes(reportStatus)) {
            await Award(prisma, reportCreatedById, languages).update("ReportEnd", 1);
            await Reputation().update("ReportWasRejected", prisma, reportCreatedById);
        }
        if (reportStatus === "ClosedDeleted") {
            if (objectOwner.__typename === "Organization") {
                const admins = await prisma.organization.findUnique({
                    where: { id: objectOwner.id },
                    select: {
                        members: {
                            select: {
                                id: true,
                                isAdmin: true,
                            },
                        },
                    },
                });
                const adminIds = admins?.members.filter(member => member.isAdmin).map(member => member.id) ?? [];
                for (const adminId of adminIds) {
                    await Reputation().update("ObjectDeletedFromReport", prisma, adminId);
                }
            }
            else {
                await Reputation().update("ObjectDeletedFromReport", prisma, objectOwner.id);
            }
        }
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushReportStatusChange(reportId, objectId, objectType, reportStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingReportId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingReportId);
            }
            notification.toSubscribers(objectType, objectId, userUpdatingReportId);
        }
    },
    runProjectComplete: async (runTitle, runId, userId) => {
        Award(prisma, userId, languages).update("RunProject", 1);
    },
    runRoutineComplete: async (runId, userId, wasAutomatic) => {
        if (wasAutomatic)
            Notify(prisma, languages).pushRunCompletedAutomatically(runId).toUser(userId);
        Award(prisma, userId, languages).update("RunRoutine", 1);
    },
    runRoutineFail: async (runId, userId, wasAutomatic) => {
        if (wasAutomatic)
            Notify(prisma, languages).pushRunFailedAutomatically(runId).toUser(userId);
    },
    runRoutineStart: async (runId, userId, wasAutomatic) => {
        if (wasAutomatic)
            Notify(prisma, languages).pushRunStartedAutomatically(runId).toUser(userId);
    },
    userInvite: async (referrerId, joinedUsername) => {
        Notify(prisma, languages).pushUserInvite(joinedUsername).toUser(referrerId);
        Award(prisma, referrerId, languages).update("UserInvite", 1);
    },
});
//# sourceMappingURL=trigger.js.map