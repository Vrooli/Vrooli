import { EventTypes, type AwardCategory, type BookmarkFor, type ChatMessage, type CopyType, type IssueStatus, type ModelType, type PullRequestStatus, type ReactionFor, type ReportStatus } from "@vrooli/shared";
import { PasswordAuthService } from "../auth/email.js";
import { DbProvider } from "../db/provider.js";
import { Notify, isObjectSubscribable } from "../notify/notify.js";
import { SocketService } from "../sockets/io.js";
import { type PreMapMessageData, type PreMapMessageDataDelete } from "../utils/messageTree.js";
import { Award, objectAwardCategory } from "./awards.js";
import { logger } from "./logger.js";
import { Reputation, objectReputationEvent, type ReputationEvent } from "./reputation.js";

export type ActionTrigger = "AccountNew" |
    "ObjectComplete" | // except runs
    "ObjectCreate" |
    "ObjectNewVersion" |
    "ObjectDelete" |
    "Fork" |
    "ObjectBookmark" |
    "ObjectReact" |
    "PullRequestClose" |
    "ReportClose" |
    "ReportContribute" |
    "RunComplete" |
    "RunStart" |
    "SessionValidate" | // for checking anniversary
    "TeamJoin" |
    "UserInvite"

type Owner = { __typename: "User" | "Team", id: string };

/**
 * Handles logging, notifications, achievements, and more when some action is performed.
 * Some examples include:
 * - Sending a push notification when someone bookmarks your comment
 * - Sending an email when your routine receives a report
 * - Giving you an award when you run a routine 100 times
 * Some actions may also do nothing right now, but it's good to send them through this function
 * in case we want to add functionality later.
 */
export function Trigger(languages: string[] | undefined) {
    return {
        accountNew: async (userId: string, userPublicId: string, emailAddress?: string) => {
            // Send a welcome/verification email (if not created with wallet)
            if (emailAddress) await PasswordAuthService.setupEmailVerificationCode(emailAddress, userId, userPublicId, languages);
            // Give the user an award
            Award(userId, languages).update("AccountNew", 1);
        },
        chatMessageCreated: async ({
            excludeUserId,
            chatId,
            messageId,
            senderId,
            message,
        }: {
            /** Adds your user ID to the push notifications omit list */
            excludeUserId: string,
            chatId: string,
            messageId: string,
            senderId: string,
            message: ChatMessage,
        }) => {
            Notify(languages).pushMessageReceived(messageId, senderId).toChatParticipants(chatId, [senderId, excludeUserId]);
            SocketService.get().emitSocketEvent(EventTypes.CHAT.MESSAGE_ADDED, chatId, {
                chatId,
                messages: [message],
            });
        },
        chatMessageUpdated: async ({
            data,
            message,
        }: {
            data: Pick<PreMapMessageData, "chatId">,
            message: ChatMessage,
        }) => {
            if (data.chatId) {
                SocketService.get().emitSocketEvent(EventTypes.CHAT.MESSAGE_UPDATED, data.chatId, {
                    chatId: data.chatId,
                    messages: [message],
                });
            } else {
                logger.error("Could not send socket event for ChatMessage", { trace: "0496", message, data });
            }
        },
        chatMessageDeleted: async ({
            data,
        }: {
            data: Pick<PreMapMessageDataDelete, "chatId" | "messageId">,
        }) => {
            if (data.chatId) {
                SocketService.get().emitSocketEvent(EventTypes.CHAT.MESSAGE_REMOVED, data.chatId, {
                    chatId: data.chatId,
                    messageIds: [data.messageId],
                });
            } else {
                logger.error("Could not send socket event for ChatMessage}", { trace: "0497", data });
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
            objectType: ModelType | `${ModelType}`,
            userUpdatingIssueId: string,
        }) => {
            // Ignore drafts
            if (issueStatus === "Draft") return;
            // If issue has never been closed or rejected (to prevent users from reopning issues to get awards/reputation)
            if (!issueHasBeenClosedOrRejected) {
                // If creating, track award progress of issue creator. 
                if (issueStatus === "Open") {
                    await Award(issueCreatedById, languages).update("IssueCreate", 1);
                }
                // If issue marked as resolved, increase reputation of issue creator
                else if (issueStatus === "ClosedResolved") {
                    await Reputation().update("IssueCreatedWasAccepted", issueCreatedById);
                }
                // If issue marked as rejected, decrease reputation of issue creator
                else if (issueStatus === "Rejected") {
                    await Reputation().update("IssueCreatedWasRejected", issueCreatedById);
                }
            }
            // Send notification
            const notification = Notify(languages).pushIssueStatusChange(issueId, objectId, objectType, issueStatus);
            notification.toAll("Issue", objectId, objectOwner, [userUpdatingIssueId]);
        },
        /**
         * Handle object creation. 
         * 
         * NOTE: Do NOT use this to handle new objects that are being transferred immediately 
         * to a user/team that's not yours
         * 
         * Steps: 
         * 1. If trackable for Awards AND the object was not copied/forked, increment progress
         * 2. If trackable for Reputation AND the object was not copied/forked, increment progress if:
         *     - Is a root object and has a public, complete version
         *     - Is a version and is public and complete
         *     - Is not a versionable object and is public and complete
         * 3. If added to team, send notification to team members
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
            objectType: `${ModelType}`,
            projectId?: string,
        }) => {
            // Step 1
            const awardCategory = objectAwardCategory(objectType);
            if (!hasParent && awardCategory) {
                await Award(createdById, languages).update(awardCategory as AwardCategory, 1);
            }
            // Step 2
            // If the object is public and complete, increase reputation score
            const reputationEvent = objectReputationEvent(objectType);
            if (!hasParent && reputationEvent && hasCompleteAndPublic) {
                await Reputation().update(reputationEvent as Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport">, createdById, objectId);
            }
            // Step 3
            // Determine if the object is subscribable
            const isSubscribable = isObjectSubscribable(objectType);
            // If the object was added to a team, send notification to team members
            if (isSubscribable && owner.__typename === "Team") {
                const notification = Notify(languages).pushNewObjectInTeam(objectType, objectId, owner.id);
                // Send notification to admins, except the user who added it
                notification.toTeam(owner.id, createdById);
                // Send notification to subscribers of the team
                notification.toSubscribers("Team", owner.id, createdById);
            }
            // Step 4
            // If the object was added to a project, send notification to project members
            if (isSubscribable && projectId) {
                const notification = Notify(languages).pushNewObjectInProject(objectType, objectId, projectId);
                // Send notification to object owner
                notification.toOwner(owner, createdById);
                // Send notification to subscribers of the project (projects are a type of resource)
                notification.toSubscribers("Resource", projectId, createdById);
            }
            // Step 5
            // If object is an email, phone, or wallet
            if (["Email", "Phone", "Wallet"].includes(objectType)) {
                // Send notification to user warning them that a new sign in method was added
                try {
                    await Notify(languages).pushNewDeviceSignIn().toUser(createdById);
                } catch (notificationError) {
                    console.warn("Failed to send new device sign-in notification:", notificationError instanceof Error ? notificationError.message : String(notificationError));
                }
            }
        },
        /**
         * Object update logic: 
         * 0. Don't do anything for Award progress, nor for adding to team. Neither of these are applicable to updates.
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
            objectType: `${ModelType}`,
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
                    await Reputation().update(reputationEvent as Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport">, updatedById, objectId);
                }
            }
            // Step 2
            // If the object has a new project
            if (projectId && projectId !== originalProjectId) {
                // Send notification to owner(s) and subscribers
                const notification = Notify(languages).pushNewObjectInProject(objectType, objectId, projectId);
                notification.toAll(objectType, objectId, owner, [updatedById]);
            }
        },
        /**
         * Object delete logic:
         * 0. Don't do anything for Award progress, nor for adding to team or project, 
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
            objectType: `${ModelType}`,
            wasCompleteAndPublic: boolean,
        }) => {
            // Step 1
            // If the object was not copied/forked (i.e. has no parent), was public and complete, and you are the original owner
            if (!hasParent && !hasBeenTransferred && wasCompleteAndPublic) {
                // If the object is trackable for reputation, decrease reputation score
                const reputationEvent = objectReputationEvent(objectType);
                if (reputationEvent) {
                    await Reputation().unCreateObject(objectId, deletedById);
                }
            }
        },
        objectCopy: async (owner: Owner, forkedByUserId: string, objectType: CopyType, parentId: string) => {
            // const notification = Notify(languages).pushObjectFork();
            // // Send notification to owner(s), depending on how many forks the object already has
            // fdfdafdsaf
            // notification.toOwner(owner, forkedByUserId);
        },
        objectBookmark: async (isBookmarked: boolean, objectType: BookmarkFor, objectId: string, userId: string) => {
            // const notification = Notify(languages).pushObjectStar();
            // // Send notification to owner(s), depending on how many bookmarks the object already has
            // fasdf
            // // Increase reputation score of object owner(s)
            // asdfasdf
        },
        objectReact: async ({
            deltaScore,
            objectType,
            objectId,
            updatedScore,
            userId,
            objectOwner,
            languages,
        }: {
            deltaScore: number;
            objectType: ReactionFor;
            objectId: string;
            updatedScore: number,
            userId: string;
            objectOwner: Owner | null | undefined,
            languages: string[];
        }) => {
            // Define score thresholds for sending notifications
            // eslint-disable-next-line no-magic-numbers
            const scoreNotifyThresolds = [1, 5, 10, 25, 50, 100, 250, 500, 1000];
            const dontNotifyFor = ["ChatMessage"];
            if (scoreNotifyThresolds.includes(updatedScore) && deltaScore > 0 && !dontNotifyFor.includes(objectType)) {
                const notification = Notify(languages).pushObjectReceivedUpvote(objectType, objectId, updatedScore);
                notification.toAll(objectType as string as ModelType, objectId, objectOwner, [userId]);
            }
            // // Increase/decrease reputation score of object owner(s), depending on sentiment of currentReaction compared to previousReaction
            // TODO
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
            objectType: ModelType | `${ModelType}`,
            userUpdatingPullRequestId: string,
        }) => {
            // Ignore drafts
            if (pullRequestStatus === "Draft") return;
            // If pullRequest has never been closed or rejected (to prevent users from reopning pullRequests to get awards/reputation)
            if (!pullRequestHasBeenClosedOrRejected) {
                // If creating, track award progress of pullRequest creator. 
                if (pullRequestStatus === "Open") {
                    await Award(pullRequestCreatedById, languages).update("PullRequestCreate", 1);
                }
                // If canceling (by creator of pull request), undo award progress of pullRequest creator
                else if (pullRequestStatus === "Canceled") {
                    await Award(pullRequestCreatedById, languages).update("PullRequestCreate", -1);
                }
                // If pullRequest marked as merged, track award and increase reputation of pullRequest creator
                else if (pullRequestStatus === "Merged") {
                    await Award(pullRequestCreatedById, languages).update("PullRequestComplete", 1);
                    await Reputation().update("PullRequestWasAccepted", pullRequestCreatedById);
                }
                // If pullRequest marked as rejected, decrease reputation of pullRequest creator
                else if (pullRequestStatus === "Rejected") {
                    await Reputation().update("PullRequestWasRejected", pullRequestCreatedById);
                }
            }
            // Send notification to object owner(s) and subscribers for the pull request and the object with the pull request
            const notification = Notify(languages).pushPullRequestStatusChange(pullRequestId, objectId, objectType, pullRequestStatus);
            notification.toAll("PullRequest", pullRequestId, { __typename: "User" as const, id: pullRequestCreatedById }, [userUpdatingPullRequestId]);
            notification.toAll(objectType, objectId, objectOwner, [userUpdatingPullRequestId]);
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
            objectType: ModelType | `${ModelType}`,
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
                    const award = await Award(contributorId, languages).update("ReportContribute", 1);
                    // Increase reputation
                    await Reputation().updateReportContribute(award.progress, contributorId);
                }
                // Increase reputation of report creator
            }
            // If report was successfull, track award progress and reputation of report creator
            if (reportCreatedById && ["ClosedDeleted", "ClosedResolved", "ClosedSuspended"].includes(reportStatus)) {
                await Award(reportCreatedById, languages).update("ReportEnd", 1);
                await Reputation().update("ReportWasAccepted", reportCreatedById);
            }
            // If report was not successfull, track award progress and reputation of report creator
            if (reportCreatedById && ["ClosedFalseReport", "ClosedNonIssue"].includes(reportStatus)) {
                await Award(reportCreatedById, languages).update("ReportEnd", 1);
                await Reputation().update("ReportWasRejected", reportCreatedById);
            }
            // If object was deleted, decrease reputation of object owner(s)
            if (reportStatus === "ClosedDeleted") {
                // If owners are a team, decrease reputation of all admins
                if (objectOwner.__typename === "Team") {
                    const admins = await DbProvider.get().team.findUnique({
                        where: { id: BigInt(objectOwner.id) },
                        select: {
                            members: {
                                select: {
                                    userId: true,
                                    isAdmin: true,
                                },
                            },
                        },
                    });
                    const adminIds = admins?.members.filter(member => member.isAdmin).map(member => member.userId.toString()) ?? [];
                    for (const adminId of adminIds) {
                        await Reputation().update("ObjectDeletedFromReport", adminId);
                    }
                }
                // Otherwise, decrease reputation of object owner
                else {
                    await Reputation().update("ObjectDeletedFromReport", objectOwner.id);
                }
            }
            // Send notification to object owner(s) and subscribers of the object with the report
            const notification = Notify(languages).pushReportStatusChange(reportId, objectId, objectType, reportStatus);
            notification.toAll(objectType, objectId, objectOwner, userUpdatingReportId ? [userUpdatingReportId] : []);
        },
        runProjectComplete: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If completed automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunCompletedAutomatically(runId).toUser(userId);
            // Track award progress
            Award(userId, languages).update("RunProject", 1);
            // If run data is public, send notification to owner of routine (depending on how many public runs the project already has)
            //Notify(languages).pushNewRunDataAvailable(runId).toOwner(asdfasdf);
        },
        runProjectFail: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If completed automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunFailedAutomatically(runId).toUser(userId);
        },
        runProjectStart: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If started automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunStartedAutomatically(runId).toUser(userId);
        },
        runRoutineComplete: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If completed automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunCompletedAutomatically(runId).toUser(userId);
            // Track award progress
            Award(userId, languages).update("RunRoutine", 1);
            // If run data is public, send notification to owner of routine (depending on how many public runs the routine already has)
            //Notify(languages).pushNewRunDataAvailable(runId).toOwner(asdfasdf);
        },
        runRoutineFail: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If completed automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunFailedAutomatically(runId).toUser(userId);
        },
        runRoutineStart: async (runId: string, userId: string, wasAutomatic: boolean) => {
            // If started automatically, send notification to user
            if (wasAutomatic) Notify(languages).pushRunStartedAutomatically(runId).toUser(userId);
        },
        teamJoin: async (teamId: string, userId: string) => {
            // const notification = Notify(languages).pushTeamJoin();
            // // Send notification to admins of team
            // asdf
        },
        userInvite: async (referrerId: string, joinedUsername: string) => {
            // Send notification to referrer
            try {
                await Notify(languages).pushUserInvite(joinedUsername).toUser(referrerId);
            } catch (notificationError) {
                console.warn("Failed to send user invite notification:", notificationError instanceof Error ? notificationError.message : String(notificationError));
            }
            // Track award progress
            Award(referrerId, languages).update("UserInvite", 1);
        },
    };
}

// AI_CHECK: TASK_ID=fix-trigger-exports | LAST: 2025-01-29
