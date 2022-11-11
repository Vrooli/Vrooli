import { DeleteOneType, ForkType, StarFor, VoteFor } from "@shared/consts";
import { ProfileModel } from "../models";
import { GraphQLModelType } from "../models/types";
import { Notify } from "../notify";
import { PrismaType } from "../types";
import { Award, AwardCategory } from "./awards";

export enum ActionTrigger {
    AccountNew = 'AccountNew',
    ObjectComplete = 'ObjectComplete', // except runs
    ObjectCreate = 'ObjectCreate',
    ObjectDelete = 'ObjectDelete',
    ObjectFork = 'Fork', //objectType, objectId, req.userId
    ObjectStar = 'ObjectStar',
    ObjectVote = 'ObjectVote',
    OrganizationJoin = 'OrganizationJoin',
    PullRequestClose = 'PullRequestClose',
    QuestionAnswer = 'QuestionAnswer',
    ReportClose = 'ReportClose',
    ReportContribute = 'ReportContribute',
    RunComplete = 'RunComplete',
    RunStart = 'RunStart',
    SessionValidate = 'SessionValidate', //req.userId (for checking anniversary)
    UserInvite = 'UserInvite',
}

/**
 * Handles logging, notifications, achievements, and more when some action is performed.
 * Some examples include:
 * - Sending a push notification when someone stars your comment
 * - Sending an email when your routine receives a report
 * - Giving you an award when you run a routine 100 times
 * Some actions may also do nothing right now, but it's good to send them through this function
 * in case we want to add functionality later.
 */
export const Trigger = (prisma: PrismaType) => ({
    /**
     * Sends a verification email and gives the user a reward
     * @param userId The new user's id
     * @param emailAddress The new user's email address, if not created from wallet
     */
    acountNew: async (userId: string, emailAddress?: string) => {
        // Send a welcome/verification email (if not created with wallet)
        if (emailAddress) await ProfileModel.verify.setupVerificationCode(emailAddress, prisma);
        // Give the user an award
        Award(prisma, userId).update('AccountNew', 1);
    },
    objectComplete: async (objectType: GraphQLModelType, objectId: string, userId: string) => {
        // Track award progress, if object is a pull request, quiz, or routine
        const completeTrackableTypes = ['PullRequest', 'Quiz', 'Routine'];
        if (completeTrackableTypes.includes(objectType)) {
            // // If routine, check if routine is a learning routine
            // asdfasd
            // Award(prisma).update(userId, AwardCategory.Com, 1);
        }
    },
    objectCreate: async (objectType: GraphQLModelType, objectId: string, userId: string) => {
        // Don't forget to handle new emails
     },
    objectDelete: async (objectType: DeleteOneType, objectId: string, userId: string) => { },
    objectFork: async (objectType: ForkType, parentId: string, userId: string) => {

    },
    objectStar: async (isStar: boolean, objectType: StarFor, objectId: string, userId: string) => { },
    objectVote: async (isUpvote: boolean | null, objectType: VoteFor, objectId: string, userId: string) => { },
    organizationJoin: async (organizationId: string, userId: string) => { },
    pullRequestClose: async (pullRequestId: string, userId: string) => { },
    questionAnswer: async (questionId: string, userId: string) => { },
    reportClose: async (reportId: string, userId: string) => { },
    reportContribute: async (reportId: string, userId: string) => { },
    runComplete: async (runId: string, userId: string, wasAutomatic: boolean, wasSuccessful: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, userId).pushRunComplete(runId);
    },
    runStart: async (runId: string, userId: string) => {
        // If started automatically, send notification to user
    },
    sessionValidate: async (userId: string) => { },
    userInvite: async (userId: string) => { },
});