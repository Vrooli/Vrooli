import { DeleteOneType, ForkType, StarFor, VoteFor } from "@shared/consts";
import { ProfileModel } from "../models";
import { GraphQLModelType } from "../models/types";
import { Notify } from "../notify";
import { PrismaType } from "../types";
import { Award, AwardCategory } from "./awards";

export type ActionTrigger = 'AccountNew' |
    'ObjectComplete' | // except runs
    'ObjectCreate' |
    'ObjectNewVersion' |
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
 * Maps GraphQLModelTypes to "Create" award types
 */
const CreateAwardTypeMap: { [key in GraphQLModelType]?: AwardCategory } = {
    // Api: 'ApiCreate',
    Comment: 'CommentCreate',
    // Issue: 'IssueCreate',
    // Note: 'NoteCreate',
    Organization: 'OrganizationCreate',
    // Post: 'PostCreate',
    Project: 'ProjectCreate',
    // PullRequest: 'PullRequestCreate',
    // Question: 'QuestionCreate',
    Routine: 'RoutineCreate',
    // SmartContract: 'SmartContractCreate',
    Standard: 'StandardCreate',
}

/**
 * Maps GraphQLModelTypes to "Complete" award types
 */
const CompleteAwardTypeMap: { [key in GraphQLModelType]?: AwardCategory } = {
    // PullRequest: 'PullRequestComplete',
    // Quiz: 'QuizPass',
    RunRoutine: 'RunRoutine',
    // RunProject: 'RunProject',
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
export const Trigger = (prisma: PrismaType, languages: string[]) => ({
    /**
     * Sends a verification email and gives the user a reward
     * @param userId The new user's id
     * @param emailAddress The new user's email address, if not created from wallet
     */
    acountNew: async (userId: string, emailAddress?: string) => {
        // Send a welcome/verification email (if not created with wallet)
        if (emailAddress) await ProfileModel.verify.setupVerificationCode(emailAddress, prisma, languages);
        // Give the user an award
        Award(prisma, userId, languages).update('AccountNew', 1);
    },
    objectComplete: async (objectType: GraphQLModelType, objectId: string, userId: string) => {
        // Track award progress
        const awardType = CompleteAwardTypeMap[objectType];
        if (awardType) Award(prisma, userId, languages).update(awardType, 1);
    },
    objectCreate: async (objectType: GraphQLModelType, objectId: string, userId: string) => {
        // If object was an email or a wallet, send notification to user warning them that a new sign in method was added
        if (['Email', 'Wallet'].includes(objectType)) {
            Notify(prisma, languages).pushNewDeviceSignIn().toUser(userId)
        }
        // If object was added to a project, send notification to anyone subscribed to the project
        asdfasfd
        // If object was created by an organization, send notification to anyone subscribed to the organization (except for who created it)
        Notify(prisma, languages).pushCreatedObject(objectName, objectType, objectId).toOrganizationSubscribers(userId);
        fdsafdsaf
        // If object was an auto-created quiz, send notification to user
        fdsafds
        // Track award progress
        const awardType = CreateAwardTypeMap[objectType];
        if (awardType) Award(prisma, userId, languages).update(awardType, 1);
    },
    objectNewVersion: async (objectType: GraphQLModelType, objectId: string, userId: string) => {
        // Send notification to anyone subscribed to the object
        asdfasdfas
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
        if (wasAutomatic) Notify(prisma, languages).pushRunCompletedAutomatically(runTitle, runId).toUser(userId);
        // Track award progress
        Award(prisma, userId, languages).update('RunComplete', 1);
    },
    runFail: async (runTitle: string, runId: string, userId: string, wasAutomatic: boolean) => {
        // If completed automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunFailedAutomatically(runTitle, runId).toUser(userId);
    },
    runStart: async (runTitle: string, runId: string, userId: string, wasAutomatic: boolean) => {
        // If started automatically, send notification to user
        if (wasAutomatic) Notify(prisma, languages).pushRunStartedAutomatically(runTitle, runId).toUser(userId);
    },
    sessionValidate: async (userId: string) => { },
    userInvite: async (userId: string) => {
        // Send notification to user
        Notify(prisma, languages).pushUserInvite(fdsaf).toUser(userId);
        // Track award progress
        Award(prisma, userId, languages).update('UserInvite', 1);
    },
});