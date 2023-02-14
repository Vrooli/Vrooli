import { IssueStatus, PullRequestStatus, ReportStatus } from "@prisma/client";
import { DeleteType, CopyType, BookmarkFor, VoteFor, GqlModelType } from "@shared/consts";
import { setupVerificationCode } from "../auth";
import { getLogic } from "../getters";
import { Notify } from "../notify";
import { PrismaType } from "../types";
import { Award } from "./awards";
import { logger } from "./logger";

export type ActionTrigger = 'AccountNew' |
    'ObjectComplete' | // except runs
    'ObjectCreate' |
    'ObjectNewVersion' |
    'ObjectDelete' |
    'Fork' |
    'ObjectBookmark' |
    'ObjectVote' |
    'OrganizationJoin' |
    'PullRequestClose' |
    'QuestionAnswer' |
    'ReportClose' |
    'ReportContribute' |
    'RunComplete' |
    'RunStart' |
    'SessionValidate' | // for checking anniversary
    'UserInvite'

type Owner = { __typename: 'User' | 'Organization', id: string };

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
        Award(prisma, userId, languages).update('AccountNew', 1);
    },
    createApi: async (userId: string, apiId: string) => {
        // // Track award progress
        // asdfasfdasdf
        // // If the version is public and complete, increase reputation score
        // adfdsfsa
    },
    deleteApiVersion: async (userId: string, apiId: string, versionId: string, versionStatus: string) => {
        // If the version was public and complete and has never been transferred, decrease reputation score
        //fdsafdsafdsa
    },
    createComment: async (userId: string, commentId: string) => {
        // Send notification to object owner(s)
        //asdfasdf
        // Track award progress
        Award(prisma, userId, languages).update('CommentCreate', 1);
    },
    createEmail: async (userId: string) => {
        // Send notification to user warning them that a new sign in method was added
        Notify(prisma, languages).pushNewDeviceSignIn().toUser(userId)
    },
    createIssue: async (issueId: string, owner: Owner, createdById: string) => {
        // Send notification to object owner(s)
        Notify(prisma, languages).pushNewIssueOnObject(issueId).toOwner(owner, createdById);
    },
    createOrganization: async (userId: string, organizationId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('OrganizationCreate', 1);
    },
    createProject: async (userId: string, projectId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('ProjectCreate', 1);
        // If the version is public and complete, increase reputation score
       // asdfdsfdsd
    },
    deleteProjectVersion: async (userId: string, projectId: string, versionId: string, versionStatus: string) => {
        //asdfasdf
    },
    createQuestion: async (userId: string, questionId: string) => {
        // Send notification to object owner(s)
        //Notify(prisma, languages).pushNewQuestion().fdsafdsafd
    },
    createRoutine: async (userId: string, routineId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('RoutineCreate', 1);
        // If the version is public and complete, increase reputation score
        //asdfdsfdsd
    },
    deleteRoutineVersion: async (userId: string, routineId: string, versionId: string, versionStatus: string) => {
        //fdsafdsafd
    },
    createSmartContract: async (userId: string, smartContractId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('SmartContractCreate', 1);
        // If the version is public and complete, increase reputation score
        //asdfdsfdsd
    },
    deleteSmartContractVersion: async (userId: string, smartContractId: string, versionId: string, versionStatus: string) => {
        //fdsafdsaf
    },
    createStandard: async (userId: string, standardId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('StandardCreate', 1);
        // If the version is public and complete, increase reputation score
        //asdfdsfdsd
    },
    deleteStandardVersion: async (userId: string, standardId: string, versionId: string, versionStatus: string) => {
       // fdsafdsaf
    },
    createWallet: async (userId: string, walletAddress: string) => {
        // Send notification to user warning them that a new sign in method was added
        Notify(prisma, languages).pushNewDeviceSignIn().toUser(userId)
    },
    issueClosed: async (owner: Owner, closedByUserId: string, issueId: string, issueStatus: IssueStatus) => {
        // Send notification to object owner(s)
       // Notify(prisma, languages).pushIssueClosed(issueId).toOwner(owner, closedByUserId);
        // Track award progress of issue creator
        //fdsafdsafds
        // If issue marked as resolved, icrease reputation of issue creator
        //fdsafdasfd
        // If issue marked as rejected, decrease reputation of issue creator
        //asdfasdf
    },
    objectAddedToOrganization: async (userId: string, organizationId: string, objectId: string, objectType: `${GqlModelType}`) => {
        // const notification = Notify(prisma, languages).pushOrganizationActivity();
        // // Send notification to admins, except the user who added it
        // notification.toOrganization(organizationId, userId);
        // // Send notification to subscribers of the organization
        // notification.toSubscribers('Organization', organizationId, userId);
    },
    objectAddedToProject: async (owner: Owner, addedByUserId: string, projectId: string, objectId: string, objectType: `${GqlModelType}`) => {
        // const notification = Notify(prisma, languages).pushProjectActivity();
        // // Send notification to object owner
        // notification.toOwner(owner, addedByUserId)
        // // Send notification to subscribers of the project
        // notification.toSubscribers('Project', projectId, addedByUserId);
    },
    objectNewVersion: async (
        updatedByUserId: string,
        objectType: `${GqlModelType}`,
        objectId: string,
        owner: Owner | null,
        hasOriginalOwner: boolean,
        wasCompleteAndPublic: boolean,
        isCompleteAndPublic: boolean,
    ) => {
        // // If the object has a complete version and is public (and wasn't before)
        // if (!wasCompleteAndPublic && isCompleteAndPublic) {
        //     // If never transferred, increase reputation score of creator
        //     if (hasOriginalOwner) {
        //         asdfasfdasfd
        //     }
        //     // Notify owners and subscribers of new version
        //     const notification = Notify(prisma, languages).pushObjectNewVersion();
        //     notification.toOwner(owner, updatedByUserId);
        //     notification.toSubscribers(objectType, objectId, updatedByUserId);
        // }
        // // If the version was public and complete, but is now not
        // else if (wasCompleteAndPublic && !isCompleteAndPublic) {
        //     // If never transferred, decrease reputation score of creator
        //     if (hasOriginalOwner) {
        //         asdfasfdasfd
        //     }
        // }
    },
    //objectDeletedVersion: fdsafdsafasfdasfdsfsa,
    /**
     * NOTE: Unless the object is soft-deleted, this must be called BEFORE the object is deleted.
     */
    objectDelete: async (owner: Owner, deletedByUserId: string, objectType: DeleteType, objectId: string) => {
        // const notification = Notify(prisma, languages).pushObjectDelete();
        // // Send notification to owner(s) (except for who deleted it)
        // notification.toOwner(owner, deletedByUserId);
        // // Send notification to anyone subscribed to the object
        // notification.toSubscribers(objectType, objectId, deletedByUserId);
    },
    objectCopy: async (owner: Owner, forkedByUserId: string, objectType: CopyType, parentId: string) => {
        // const notification = Notify(prisma, languages).pushObjectFork();
        // // Send notification to owner(s), depending on how many forks the object already has
        // fdfdafdsaf
        // notification.toOwner(owner, forkedByUserId);
    },
    objectStar: async (isBookmarked: boolean, objectType: BookmarkFor, objectId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushObjectStar();
        // // Send notification to owner(s), depending on how many bookmarks the object already has
        // fasdf
        // // Increase reputation score of object owner(s)
        // asdfasdf
    },
    objectVote: async (isUpvote: boolean | null, objectType: VoteFor, objectId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushObjectVote();
        // // Send notification to owner(s), depending on how many votes the object already has
        // asdf
        // // Increase reputation score of object owner(s)
        // asdfasdf
    },
    organizationJoin: async (organizationId: string, userId: string) => {
        // const notification = Notify(prisma, languages).pushOrganizationJoin();
        // // Send notification to admins of organization
        // asdf
    },
    pullRequestClose: async (objectType: `${GqlModelType}`, objectId: string, status: PullRequestStatus, userId: string) => {
        // // If pull request was accepted, increase award progress and reputation of pull request creator
        // asdf
        // // If pull request was rejected, decrease reputation of pull request creator
        // asdf
        // const notification = Notify(prisma, languages).pushPullRequestClose();
        // // Send notification to owner(s) (except for who created it)
        // asdfasfdsf
        // // Send notifications to subscribers
        // asdfasdfas
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
    reportClose: async (reportId: string, userId: string, status: ReportStatus) => {
        // // Send notification to creator of report
        // asdf
        // // Send notification to anyone subscribed to the report
        // asdf
        // // Track award progress and reputation for anyone who contributed to the report
        // asdf
        // // If report was accepted, increase reputation of user who created the report
        // asdf
        // // If report was rejected, decrease reputation of user who created the report
        // asdf
        // // If report resulted in object being deleted, reduce reputation of owner
        // asdf
    },
    reportOpen: async (reportId: string, userId: string) => {
        // // Send notification to owner(s) of object
        // asdfasfdsafds
    },
    runProjectComplete: async (runTitle: string, runId: string, userId: string) => {
        // Track award progress
        Award(prisma, userId, languages).update('RunProject', 1);
        // If run data is public, send notification to owner of routine (depending on how many public runs the project already has)
        //Notify(prisma, languages).pushNewRunDataAvailable(runId).toOwner(asdfasdf);
    },
    runRoutineComplete: async (runId: string, userId: string, wasAutomatic: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunCompletedAutomatically(runId).toUser(userId);
        // Track award progress
        Award(prisma, userId, languages).update('RunRoutine', 1);
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
        Award(prisma, referrerId, languages).update('UserInvite', 1);
    },
});