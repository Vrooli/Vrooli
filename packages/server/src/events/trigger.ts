import { DeleteOneType, ForkType, StarFor, VoteFor } from "@shared/consts";
import { ProfileModel } from "../models";
import { GraphQLModelType } from "../models/types";
import { Notify } from "../notify";
import { PrismaType } from "../types";
import { Award } from "./awards";

export type ActionTrigger = 'AccountNew' |
    'ObjectComplete' | // except runs
    'ObjectCreate' |
    'ObjectDelete' |
    'Fork' |
    'ObjectStar' |
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
    runComplete: async (runTitle: string, runId: string, userId: string, wasAutomatic: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, userId).pushRunComplete(runTitle, runId);
        // Track award progress
        Award(prisma, userId).update('RunComplete', 1);
    },
    runFail: async (runTitle: string, runId: string, userId: string, wasAutomatic: boolean) => { 
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, userId).pushRunFail(runTitle, runId);
    },
    runStart: async (runTitle: string, runId: string, userId: string, wasAutomatic: boolean) => {
        // If started automatically, send notification to user
        if (wasAutomatic) Notify(prisma, userId).pushRunStartedAutomatically(runTitle, runId);
    },
    sessionValidate: async (userId: string) => { },
    userInvite: async (userId: string) => { },
});