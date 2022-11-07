import { PrismaType } from "../types";

export type NotificationUrgency = 'low' | 'normal' | 'critical';

export enum NotificationCategory {
    AccountCreditsOrApi = 'AccountCreditsOrApi',
    Award = 'Award',
    IssueActivity = 'IssueActivity',
    NewQuestionOrIssue = 'NewQuestionOrIssue',
    ObjectStarVoteFork = 'ObjectStarVoteFork',
    ObjectDelete = 'ObjectDelete',
    Promotion = 'Promotion',
    PullRequestClose = 'PullRequestClose',
    QuestionActivty = 'QuestionActivity',
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
    /**
     * Base function for pushing a notification to a user. If the user 
     * has no devices, emails, or phone numbers to send the message to, 
     * it is still stored in the database for the user to see when they 
     * open the app.
     */
    push: async (category: NotificationCategory, title: string, body: string, data: any) => {
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
    },
    /**
     * Sends an AccountCreditsOrApi notification
     */
    pushAccountCreditsOrApi: async () => {
        //TODO
    },
    /**
     * Sends an Award notification
     */
    pushAward: async () => {
        //TODO
    },
    pushIssueActivity: async () => { },
    pushNewQuestionOrIssue: async () => { },
    pushObjectStarVoteFork: async () => { },
    pushObjectDelete: async () => { },
    pushPromotion: async () => { },
    pushPullRequestClose: async () => { },
    pushQuestionActivity: async () => { },
    pushRun: async () => { },
    pushSchedule: async () => { },
    pushSecurity: async () => { },
    pushStreak: async () => { },
    pushUserInvite: async () => { },
});