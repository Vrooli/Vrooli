import { PrismaType } from "../types";

export type NotificationUrgency = 'low' | 'normal' | 'critical';

export enum NotificationCategory {
    AccountCreditsOrApi = 'AccountCreditsOrApi',
    Award = 'Award',
    IssueActivity = 'IssueActivity',
    NewQuestionOrIssue = 'NewQuestionOrIssue',
    ObjectStarVoteFork = 'ObjectStarVoteFork',
    Promotion = 'Promotion',
    PullRequestClose = 'PullRequestClose',
    QuestionActivty = 'QuestionActivity',
    ReportClose = 'ReportClose',
    Run = 'Run',
    Schedule = 'Schedule',
    Security = 'Security',
    Streak = 'Streak',
    UserInvite = 'UserInvite',
}

/**
 * Queries a user for their notification permissions and push devices, emails, and phone numbers
 * @param userId The user's id
 */
const getNotificationSettingsAndDevices = async (userId: string): Promise<{ settings: any, pushDevices: any, emails: any, phoneNumbers: any }> => {
    //TODO
    return {} as any;
}

type PushParams = {
    body: string,
    category: NotificationCategory,
    link?: string,
    prisma: PrismaType,
    title?: string,
    userId: string,
}

/**
 * Base function for pushing a notification to a user. If the user 
 * has no devices, emails, or phone numbers to send the message to, 
 * it is still stored in the database for the user to see when they 
 * open the app.
 */
const push = async ({ body, category, link, prisma, title, userId }: PushParams) => {
    // Get the user's notification settings and devices
    const { settings, pushDevices, emails, phoneNumbers } = await getNotificationSettingsAndDevices(userId);
    // Filter out devices which don't allow any notifications
    // TODO
    // Increment count in Redis for this user. If it is over the limit, don't send the notification
    // TODO
    // Send the notification to each allowed device
    //TODO
    // Store the notification in the database
    //TODO
}

/**
 * Creates an appropriate label for a near-future event. 
 * Examples: (5 minutes, 2 hours, 1 day)
 * @param date The date of the event. If it's in the past, we return "now"
 */
const getEventStartLabel = (date: Date) => {
    const now = new Date();
    const diff = date.getTime() - now.getTime();
    if (diff < 0) return 'now';
    if (diff < 1000 * 60) return 'in a few seconds';
    if (diff < 1000 * 60 * 60) return `in ${Math.round(diff / (1000 * 60))} minutes`;
    if (diff < 1000 * 60 * 60 * 24) return `in ${Math.round(diff / (1000 * 60 * 60))} hours`;
    return `in ${Math.round(diff / (1000 * 60 * 60 * 24))} days`;
}

/**
 * Handles sending and registering notifications for a user. 
 * Notifications settings and devices are queried from the main database.
 * Notification limits are tracked using Redis.
 */
 export const Notify = (prisma: PrismaType, userId: string) => ({
    /**
     * Sets up a push device to receive notifications
     * @param userId The user's id
     * @param device The device to register
     */
    registerPushDevice: async (device: any) => {
        //TODO
    },
    /**
     * Updates a user's notification settings
     * @param settings The new settings
     * @param emails Email addresses with updated notification settings
     * @param phoneNumbers Phone numbers with updated notification settings
     */
    updateSettings: async (settings: any, emails: any, phoneNumbers: any) => {
        //TODO
    },
    pushApiOutOfCredits: async () => {
        const title = 'API out of credits';
        const body = 'Your API ran out of credits. Please add more credits to your account if you want to continue using it, or wait until tomorrow to receive new credits.';
        await push({ body, category: NotificationCategory.AccountCreditsOrApi, prisma, title, userId });
    },
    pushAward: async (awardName: string, awardDescription: string) => {
        const title = 'New Award Earned';
        const body = `You earned the "${awardName}" award! ${awardDescription}`;
        const link = `/awards`;
        push({ body, category: NotificationCategory.Award, link, prisma, title, userId });
    },
    pushIssueActivity: async (issueName: string, issueId: string) => { 
        const title = 'Issue Activity';
        const body = `New activity on issue "${issueName}"`;
        const link = `/issues/${issueId}`;
        push({ body, category: NotificationCategory.IssueActivity, link, prisma, title, userId });
    },
    pushNewQuestionOnObject: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'New Question';
        const body = `There's a new Question on ${objectName}. Do you know the answer?`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.NewQuestionOrIssue, link, prisma, title, userId });
    },
    pushNewIssueOnObject: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'New Issue';
        const body = `There's a new Issue on ${objectName}. Can you help?`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.NewQuestionOrIssue, link, prisma, title, userId });
    },
    pushObjectReceivedStar: async (objectName: string, objectType: string, objectId: string, totalStars: number) => {
        const title = 'New Stars';
        const body = `${objectName} now has ${totalStars} ${totalStars === 1 ? 'star' : 'stars'}!`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.ObjectStarVoteFork, link, prisma, title, userId });
    },
    pushObjectReceivedUpvote: async (objectName: string, objectType: string, objectId: string, totalScore: number) => {
        const title = 'New Votes';
        const body = `${objectName} now has a score of ${totalScore}!`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.ObjectStarVoteFork, link, prisma, title, userId });
    },
    pushObjectReceivedFork: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'New Fork';
        const body = `${objectName} was forked!`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.ObjectStarVoteFork, link, prisma, title, userId });
    },
    pushReportClosedObjectDeleted: async (objectName: string) => {
        const title = 'Object Deleted';
        const body = `Sorry, the community decided to delete ${objectName}. Your reputation score has been reduced.`;
        push({ body, category: NotificationCategory.ReportClose, prisma, title, userId });
    },
    pushReportClosedObjectHidden: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'Object Hidden';
        const body = `${objectName} has been hidden from the community. Please fix the issues mentioned in its reports and publish an updated version if you'd like it to be visible again.`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.ReportClose, link, prisma, title, userId });
    },
    pushReportClosedAccountSuspended: async (objectName: string) => {
        const title = 'Account Suspended';
        const body = `Your account has been suspended due to the reports on ${objectName}. Please contact us if you think this was a mistake.`;
        push({ body, category: NotificationCategory.ReportClose, prisma, title, userId });
    },
    pushPromotion: async (title: string, body: string, link: string) => {
        push({ body, category: NotificationCategory.Promotion, link, prisma, title, userId });
    },
    pushPullRequestAccepted: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'Pull Request Accepted';
        const body = `Your improvements for ${objectName} have been accepted! Thank you for your contribution.`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.PullRequestClose, link, prisma, title, userId });
    },
    pushPullRequestRejected: async (objectName: string, objectType: string, objectId: string) => {
        const title = 'Pull Request Rejected';
        const body = `Sorry, your improvements for ${objectName} have been rejected.`;
        const link = `/${objectType}/${objectId}`;
        push({ body, category: NotificationCategory.PullRequestClose, link, prisma, title, userId });
    },
    pushQuestionActivity: async (questionName: string, questionId: string) => {
        const title = 'Question Activity';
        const body = `New activity on question "${questionName}"`;
        const link = `/questions/${questionId}`;
        push({ body, category: NotificationCategory.QuestionActivty, link, prisma, title, userId });
    },
    pushRunStartedAutomatically: async (runName: string, runId: string) => {
        const title = 'Run Started';
        const body = `Your run "${runName}" has started! We'll let you know when it's finished.`;
        const link = `/runs/${runId}`;
        push({ body, category: NotificationCategory.Run, link, prisma, title, userId });
    },
    pushRunComplete: async (runTitle: string, runId: string) => {
        const title = `Run completed!`;
        const body = `Your run "${runTitle}" is complete! Press here to view the results.`;
        const link = `/runs/${runId}`;
        push({ body, category: NotificationCategory.Run, link, prisma, title, userId });
    },
    pushRunFail: async (runTitle: string, runId: string) => {
        const title = `Run failed!`;
        const body = `Your run "${runTitle}" failed. Press here to see more details.`;
        const link = `/runs/${runId}`;
        push({ body, category: NotificationCategory.Run, link, prisma, title, userId });
    },
    pushScheduleOrganization: async (meetingName: string, startTime: Date) => {
        const startLabel = getEventStartLabel(startTime);
        const title = 'Meeting reminder';
        const body = `${meetingName} is starting in ${startLabel}!`;
        const link = `/meeting/${meetingName}`; //TODO
        push({ body, category: NotificationCategory.Schedule, link, prisma, title, userId });
    },
    pushScheduleUser: async (title: string, startTime: Date) => {
        const startLabel = getEventStartLabel(startTime);
        const body = `${title} is starting in ${startLabel}!`;
        push({ body, category: NotificationCategory.Schedule, prisma, userId });
    },
    pushNewDeviceSignIn: async () => { 
        const title = 'New device sign-in';
        const body = 'You have signed in to your account on a new device. If this was not you, please change your password.';
        push({ body, category: NotificationCategory.Security, prisma, title, userId });
    },
    pushStreakReminder: async (timeToReset: Date) => {
        const startLabel = getEventStartLabel(timeToReset);
        const title = 'Streak reminder';
        const body = `Your streak will be broken in ${startLabel}. Complete a routine to keep your streak going!`;
        push({ body, category: NotificationCategory.Streak, prisma, title, userId });
    },
    pushStreakBroken: async () => {
        const title = 'Streak broken';
        const body = 'Your streak has been broken. Complete a routine to start a new streak!';
        push({ body, category: NotificationCategory.Streak, prisma, title, userId });
    },
    pushUserInvite: async (friendUsername: string) => { 
        const body = `${friendUsername} has created an account using your invite code! Enjoy your free month of premium😎`;
        push({ body, category: NotificationCategory.UserInvite, prisma, userId });
    },
});