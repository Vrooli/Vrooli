import { AwardCategory, BookmarkFor, ChatMessage, CopyType, GqlModelType, IssueStatus, PullRequestStatus, ReactionFor, ReportStatus, SubscribableObject } from "@local/shared";
import { setupVerificationCode } from "../auth/email";
import { ChatMessageBeforeDeletedData, PreMapMessageData } from "../models/base/chatMessage";
import { Notify, isObjectSubscribable } from "../notify";
import { emitSocketEvent } from "../sockets/events";
import { PrismaType } from "../types";
import { Award, objectAwardCategory } from "./awards";
import { logger } from "./logger";
import { Reputation, objectReputationEvent } from "./reputation";

export type ActionTrigger = "AccountNew" |
    "ObjectComplete" | // except runs
    "ObjectCreate" |
    "ObjectNewVersion" |
    "ObjectDelete" |
    "Fork" |
    "ObjectBookmark" |
    "ObjectReact" |
    "OrganizationJoin" |
    "PullRequestClose" |
    "QuestionAnswer" |
    "ReportClose" |
    "ReportContribute" |
    "RunComplete" |
    "RunStart" |
    "SessionValidate" | // for checking anniversary
    "UserInvite"

type Owner = { __typename: "User" | "Organization", id: string };

/**
 * Handles logging, notifications, achievements, and more when some action is performed.
 * Some examples include:
 * - Sending a push notification when someone bookmarks your comment
 * - Sending an email when your routine receives a report
 * - Giving you an award when you run a routine 100 times
 * Some actions may also do nothing right now, but it's good to send them through this function
 * in case we want to add functionality later.
 */
export const Trigger = (prisma: PrismaType, languages: string[]) => ({
    acountNew: async (userId: string, emailAddress?: string) => {
        // Send a welcome/verification email (if not created with wallet)
        if (emailAddress) await setupVerificationCode(emailAddress, prisma, languages);
        // Give the user an award
        Award(prisma, userId, languages).update("AccountNew", 1);
    },
    chatMessageCreated: async ({
        createdById,
        data,
        message,
    }: {
        createdById: string,
        data: PreMapMessageData,
        message: ChatMessage,
    }) => {
        if (message.id && data.chatId) {
            Notify(prisma, languages).pushMessageReceived(message.id, data.userId).toChatParticipants(data.chatId, createdById);
            emitSocketEvent("messages", data.chatId, { added: [message] });
        } else {
            logger.error("Could not send notification or socket event for ChatMessage", { trace: "0494", message, data });
        }
    },
    chatMessageUpdated: async ({
        data,
        message,
    }: {
        data: PreMapMessageData,
        message: ChatMessage,
    }) => {
        if (data.chatId) {
            emitSocketEvent("messages", data.chatId, { edited: [message] });
        } else {
            logger.error("Could not send socket event for ChatMessage", { trace: "0496", message, data });
        }
    },
    chatMessageDeleted: async ({
        data,
        messageId,
    }: {
        data: ChatMessageBeforeDeletedData,
        messageId: string,
    }) => {
        if (data.chatId) {
            emitSocketEvent("messages", data.chatId, { deleted: [messageId] });
        } else {
            logger.error("Could not send socket event for ChatMessage}", { trace: "0497", messageId, data });
        }
    },
    issueActivity: async ({
        issueCreatedById,
        issueHasBeenClosedOrRejected,
        issueId,
        issueStatus,
        objectId,
        objectOwner,
        objectType,
        userUpdatingIssueId,
    }: {
        issueCreatedById: string,
        issueHasBeenClosedOrRejected: boolean,
        issueId: string,
        issueStatus: IssueStatus,
        objectId: string,
        objectOwner: Owner,
        objectType: GqlModelType | `${GqlModelType}`,
        userUpdatingIssueId: string,
    }) => {
        // If issue has never been closed or rejected (to prevent users from reopning issues to get awards/reputation)
        if (!issueHasBeenClosedOrRejected) {
            // If creating, track award progress of issue creator. 
            if (issueStatus === "Open") {
                await Award(prisma, issueCreatedById, languages).update("IssueCreate", 1);
            }
            // If issue marked as resolved, increase reputation of issue creator
            else if (issueStatus === "ClosedResolved") {
                await Reputation().update("IssueCreatedWasAccepted", prisma, issueCreatedById);
            }
            // If issue marked as rejected, decrease reputation of issue creator
            else if (issueStatus === "Rejected") {
                await Reputation().update("IssueCreatedWasRejected", prisma, issueCreatedById);
            }
        }
        // Send notification to object owner(s) and subscribers of both the organization and the object with the issue
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushIssueStatusChange(issueId, objectId, objectType, issueStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingIssueId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingIssueId);
            }
            notification.toSubscribers(objectType as SubscribableObject, objectId, userUpdatingIssueId);
        }
    },
    /**
     * Handle object creation. 
     * 
     * NOTE: Do NOT use this to handle new objects that are being transferred immediately 
     * to a user/organization that's not yours
     * 
     * Steps: 
     * 1. If trackable for Awards AND the object was not copied/forked, increment progress
     * 2. If trackable for Reputation AND the object was not copied/forked, increment progress if:
     *     - Is a root object and has a public, complete version
     *     - Is a version and is public and complete
     *     - Is not a versionable object and is public and complete
     * 3. If added to organization, send notification to organization members
     * 4. If added to project, send notification to project members
     * 5. Handle object-specific cases
     */
    objectCreated: async ({
        createdById,
        hasCompleteAndPublic,
        hasParent,
        owner,
        objectId,
        objectType,
        projectId,
    }: {
        createdById: string,
        hasCompleteAndPublic: boolean,
        hasParent: boolean,
        owner: Owner,
        objectId: string,
        objectType: `${GqlModelType}`,
        projectId?: string,
    }) => {
        // Step 1
        const awardCategory = objectAwardCategory(objectType);
        if (!hasParent && awardCategory) {
            await Award(prisma, createdById, languages).update(awardCategory as AwardCategory, 1);
        }
        // Step 2
        // If the object is public and complete, increase reputation score
        const reputationEvent = objectReputationEvent(objectType);
        if (!hasParent && reputationEvent && hasCompleteAndPublic) {
            await Reputation().update(reputationEvent as any, prisma, createdById, objectId);
        }
        // Step 3
        // Determine if the object is subscribable
        const isSubscribable = isObjectSubscribable(objectType);
        // If the object was added to an organization, send notification to organization members
        if (isSubscribable && owner.__typename === "Organization") {
            const notification = Notify(prisma, languages).pushNewObjectInOrganization(objectType, objectId, owner.id);
            // Send notification to admins, except the user who added it
            notification.toOrganization(owner.id, createdById);
            // Send notification to subscribers of the organization
            notification.toSubscribers("Organization", owner.id, createdById);
        }
        // Step 4
        // If the object was added to a project, send notification to project members
        if (isSubscribable && projectId) {
            const notification = Notify(prisma, languages).pushNewObjectInProject(objectType, objectId, projectId);
            // Send notification to object owner
            notification.toOwner(owner, createdById);
            // Send notification to subscribers of the project
            notification.toSubscribers("Project", projectId, createdById);
        }
        // Step 5
        // If object is an email, phone, or wallet
        if (["Email", "Phone", "Wallet"].includes(objectType)) {
            // Send notification to user warning them that a new sign in method was added
            Notify(prisma, languages).pushNewDeviceSignIn().toUser(createdById);
        }
    },
    /**
     * Object update logic: 
     * 0. Don't do anything for Award progress, nor for adding to organization. Neither of these are applicable to updates.
     * 1. If trackable for Reputation AND the object was not copied/forked, increment progress if:
     *    - Is a root object, had no public and complete version before, and now has a public, complete version
     *    - Is a version, is public and complete, and there were no public and complete versions before. 
     *    In this case, make sure that this is done only once per root object (since you could theoretically 
     *    complete multiple versions in one mutation)
     *    - Is not a versionable object, is public and complete, and was not public and complete before
     * 2. If added to a project that it wasn't in before, send notification to project members
     * 3. Object-specific cases
     */
    objectUpdated: async ({
        updatedById,
        hasCompleteAndPublic,
        hasParent,
        owner,
        objectId,
        objectType,
        originalProjectId,
        projectId,
        wasCompleteAndPublic,
    }: {
        updatedById: string,
        hasCompleteAndPublic: boolean,
        hasParent: boolean,
        owner: Owner,
        objectId: string,
        objectType: `${GqlModelType}`,
        originalProjectId?: string,
        projectId?: string,
        wasCompleteAndPublic: boolean,
    }) => {
        // Step 1
        // If the object was not copied/forked (i.e. has no parent), was not public and complete before, and is now public and complete
        if (!hasParent && !wasCompleteAndPublic && hasCompleteAndPublic) {
            // If the object is trackable for reputation, increase reputation score
            const reputationEvent = objectReputationEvent(objectType);
            if (reputationEvent) {
                await Reputation().update(reputationEvent as any, prisma, updatedById, objectId);
            }
        }
        // Step 2
        // If the object has a new project
        if (projectId && projectId !== originalProjectId) {
            // If the object is subscribable, send notification to project members
            const isSubscribable = isObjectSubscribable(objectType);
            if (isSubscribable) {
                const notification = Notify(prisma, languages).pushNewObjectInProject(objectType, objectId, projectId);
                // Send notification to object owner
                notification.toOwner(owner, updatedById);
                // Send notification to subscribers of the project
                notification.toSubscribers("Project", projectId, updatedById);
            }
        }
    },
    /**
     * Object delete logic:
     * 0. Don't do anything for Award progress, nor for adding to organization or project, 
     * nor for notifying subscribers. None of these are applicable to deletes.
     * 1. If trackable for Reputation AND the object was not copied/forked AND you are the original owner, decrement progress if:
     *    - Is a root object and had at least one public, complete version
     *    - Is a version and is public and complete, and there are no other public and complete versions
     *    - Is not a versionable object and is public and complete
     * 2. Object-specific cases
     */
    objectDeleted: async ({
        deletedById,
        hasBeenTransferred,
        hasParent,
        objectId,
        objectType,
        wasCompleteAndPublic,
    }: {
        deletedById: string,
        hasBeenTransferred: boolean,
        hasParent: boolean,
        objectId: string,
        objectType: `${GqlModelType}`,
        wasCompleteAndPublic: boolean,
    }) => {
        // Step 1
        // If the object was not copied/forked (i.e. has no parent), was public and complete, and you are the original owner
        if (!hasParent && !hasBeenTransferred && wasCompleteAndPublic) {
            // If the object is trackable for reputation, decrease reputation score
            const reputationEvent = objectReputationEvent(objectType);
            if (reputationEvent) {
                await Reputation().unCreateObject(prisma, objectId, deletedById);
            }
        }
    },
    objectCopy: async (owner: Owner, forkedByUserId: string, objectType: CopyType, parentId: string) => {
        // const notification = Notify(prisma, languages).pushObjectFork();
        // // Send notification to owner(s), depending on how many forks the object already has
        // fdfdafdsaf
        // notification.toOwner(owner, forkedByUserId);
    },
    objectBookmark: async (isBookmarked: boolean, objectType: BookmarkFor, objectId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushObjectStar();
        // // Send notification to owner(s), depending on how many bookmarks the object already has
        // fasdf
        // // Increase reputation score of object owner(s)
        // asdfasdf
    },
    objectReact: async (previousReaction: string | null | undefined, currentReaction: string | null | undefined, objectType: ReactionFor, objectId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushObjectReact();
        // // If previousReaction is null, Send notification to owner(s), depending on how many votes the object already has
        // asdf
        // // Increase/decrease reputation score of object owner(s), depending on sentiment of currentReaction compared to previousReaction
        // asdfasdf
        //TODO if reacted on chat message, send io addReaction event. Also make sure ChatCrud handles it
    },
    organizationJoin: async (organizationId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushOrganizationJoin();
        // // Send notification to admins of organization
        // asdf
    },
    /**
     * Call this any time a pull request's status changes, including when it is first created.
     */
    pullRequestActivity: async ({
        pullRequestCreatedById,
        pullRequestHasBeenClosedOrRejected,
        pullRequestId,
        pullRequestStatus,
        objectId,
        objectOwner,
        objectType,
        userUpdatingPullRequestId,
    }: {
        pullRequestCreatedById: string,
        pullRequestHasBeenClosedOrRejected: boolean,
        pullRequestId: string,
        pullRequestStatus: PullRequestStatus,
        objectId: string,
        objectOwner: Owner,
        objectType: GqlModelType | `${GqlModelType}`,
        userUpdatingPullRequestId: string,
    }) => {
        // If pullRequest has never been closed or rejected (to prevent users from reopning pullRequests to get awards/reputation)
        if (!pullRequestHasBeenClosedOrRejected) {
            // If creating, track award progress of pullRequest creator. 
            if (pullRequestStatus === "Open") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestCreate", 1);
            }
            // If canceling (by creator of pull request), undo award progress of pullRequest creator
            else if (pullRequestStatus === "Canceled") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestCreate", -1);
            }
            // If pullRequest marked as merged, track award and increase reputation of pullRequest creator
            else if (pullRequestStatus === "Merged") {
                await Award(prisma, pullRequestCreatedById, languages).update("PullRequestComplete", 1);
                await Reputation().update("PullRequestWasAccepted", prisma, pullRequestCreatedById);
            }
            // If pullRequest marked as rejected, decrease reputation of pullRequest creator
            else if (pullRequestStatus === "Rejected") {
                await Reputation().update("PullRequestWasRejected", prisma, pullRequestCreatedById);
            }
        }
        // Send notification to object owner(s) and subscribers of both the organization and the object with the pullRequest
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushPullRequestStatusChange(pullRequestId, objectId, objectType, pullRequestStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingPullRequestId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingPullRequestId);
            }
            notification.toSubscribers(objectType as SubscribableObject, objectId, userUpdatingPullRequestId);
        }
    },
    questionAccepted: async (questionId: string, answerId: string, userId: string) => {
        // // Increase award progress and reputation of answer creator
        // asdf
        // const notification = Notify(prisma, languages).pushQuestionAccepted();
        // // Send notification to answer creator
        // asdf
        // // Send notification to subscribers
        // asdf
    },
    questionAnswer: async (questionId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushQuestionAnswer();
        // // Send notification to question owner
        // asdf
        // // Send notification to anyone subscribed to the question
        // asdf
        // // Track award progress
        // Award(prisma, userId, languages).update('QuestionAnswer', 1);
        // // Increase reputation of answer creator
        // fdsafsafdsa
    },
    /**
     * Call this any time a report's status changes, including when it is first created.
     * 
     * NOTE 1: Unlike issues and pull requests, reports can never be reopened. 
     * So we don't need to worry about users gaming awards/reputation from reopening reports.
     * 
     * NOTE 2: If object is being deleted, this should be called before the object is deleted. 
     * Otherwise, we cannot display the name of the object in the notification.
     */
    reportActivity: async ({
        objectId,
        objectOwner,
        objectType,
        reportContributors,
        reportCreatedById,
        reportId,
        reportStatus,
        userUpdatingReportId,
    }: {
        objectId: string,
        objectOwner: Owner,
        objectType: GqlModelType | `${GqlModelType}`,
        reportContributors: string[],
        reportCreatedById: string | null,
        reportId: string,
        reportStatus: ReportStatus,
        userUpdatingReportId: string | null,
    }) => {
        // If report was closed, track award progress and reputation of contributors 
        // (those who responded to the report)
        if (reportStatus !== "Open") {
            // Make sure each contributor is only counted once (even though this should already be the case)
            const contributors = [...new Set(reportContributors)];
            for (const contributorId of contributors) {
                // Track award progress
                const award = await Award(prisma, contributorId, languages).update("ReportContribute", 1);
                // Increase reputation
                await Reputation().updateReportContribute(award.progress, prisma, contributorId);
            }
            // Increase reputation of report creator
        }
        // If report was successfull, track award progress and reputation of report creator
        if (reportCreatedById && ["ClosedDeleted", "ClosedResolved", "ClosedSuspended"].includes(reportStatus)) {
            await Award(prisma, reportCreatedById, languages).update("ReportEnd", 1);
            await Reputation().update("ReportWasAccepted", prisma, reportCreatedById);
        }
        // If report was not successfull, track award progress and reputation of report creator
        if (reportCreatedById && ["ClosedFalseReport", "ClosedNonIssue"].includes(reportStatus)) {
            await Award(prisma, reportCreatedById, languages).update("ReportEnd", 1);
            await Reputation().update("ReportWasRejected", prisma, reportCreatedById);
        }
        // If object was deleted, decrease reputation of object owner(s)
        if (reportStatus === "ClosedDeleted") {
            // If owners are an organization, decrease reputation of all admins
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
            // Otherwise, decrease reputation of object owner
            else {
                await Reputation().update("ObjectDeletedFromReport", prisma, objectOwner.id);
            }
        }
        // Send notification to object owner(s) and subscribers of both the organization and the object with the report
        const isSubscribable = isObjectSubscribable(objectType);
        if (isSubscribable) {
            const notification = Notify(prisma, languages).pushReportStatusChange(reportId, objectId, objectType, reportStatus);
            if (objectOwner.__typename === "Organization") {
                notification.toOrganization(objectOwner.id, userUpdatingReportId);
                notification.toSubscribers("Organization", objectOwner.id, userUpdatingReportId);
            }
            notification.toSubscribers(objectType as SubscribableObject, objectId, userUpdatingReportId);
        }
    },
    runProjectComplete: async (runTitle: string, runId: string, userId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update("RunProject", 1);
        // If run data is public, send notification to owner of routine (depending on how many public runs the project already has)
        //Notify(prisma, languages).pushNewRunDataAvailable(runId).toOwner(asdfasdf);
    },
    runRoutineComplete: async (runId: string, userId: string, wasAutomatic: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunCompletedAutomatically(runId).toUser(userId);
        // Track award progress
        Award(prisma, userId, languages).update("RunRoutine", 1);
        // If run data is public, send notification to owner of routine (depending on how many public runs the routine already has)
        //Notify(prisma, languages).pushNewRunDataAvailable(runId).toOwner(asdfasdf);
    },
    runRoutineFail: async (runId: string, userId: string, wasAutomatic: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunFailedAutomatically(runId).toUser(userId);
    },
    runRoutineStart: async (runId: string, userId: string, wasAutomatic: boolean) => {
        // If started automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunStartedAutomatically(runId).toUser(userId);
    },
    userInvite: async (referrerId: string, joinedUsername: string) => {
        // Send notification to referrer
        Notify(prisma, languages).pushUserInvite(joinedUsername).toUser(referrerId);
        // Track award progress
        Award(prisma, referrerId, languages).update("UserInvite", 1);
    },
});
