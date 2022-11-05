import { ProfileModel } from "../models";
import { PrismaType } from "../types";
import { Award, AwardCategory } from "./awards";

export enum ActionTrigger {
    AccountNew = 'AccountNew',
    ObjectComplete = 'ObjectComplete',
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
        Award(prisma).update(userId, AwardCategory.AccountNew, 1);
    },
    objectComplete: async (objectType: string, objectId: string, userId: string) => { },
    objectCreate: async (objectType: string, objectId: string, userId: string) => { },
    objectDelete: async (objectType: string, objectId: string, userId: string) => { },
    objectFork: async (objectType: string, parentId: string, userId: string) => { },
    objectStar: async (objectType: string, objectId: string, userId: string) => { },
    objectVote: async (objectType: string, objectId: string, userId: string) => { },
    organizationJoin: async (organizationId: string, userId: string) => { },
    pullRequestClose: async (pullRequestId: string, userId: string) => { },
    questionAnswer: async (questionId: string, userId: string) => { },
    reportClose: async (reportId: string, userId: string) => { },
    reportContribute: async (reportId: string, userId: string) => { },
    sessionValidate: async (userId: string) => { },
    userInvite: async (userId: string) => { },
});