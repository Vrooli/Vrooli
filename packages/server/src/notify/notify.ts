import { CustomError, logger } from "../events";
import { initializeRedis } from "../redisConn";
import { PrismaType } from "../types";
import { sendMail } from "./email";
import { findRecipientsAndLimit, NotificationSettings, updateNotificationSettings } from "./notificationSettings";
import { sendPush } from "./push";
import i18next, { TFuncKey } from 'i18next';

export type NotificationUrgency = 'low' | 'normal' | 'critical';

export type NotificationCategory = 'AccountCreditsOrApi' |
    'Award' |
    'IssueActivity' |
    'NewQuestionOrIssue' |
    'ObjectStarVoteFork' |
    'Promotion' |
    'PullRequestClose' |
    'QuestionActivity' |
    'ReportClose' |
    'Run' |
    'Schedule' |
    'Security' |
    'Streak' |
    'Transfer' |
    'UserInvite';

type TransKey = TFuncKey<'notify', undefined>

type PushParams = {
    bodyKey: TransKey,
    bodyVariables?: { [key: string]: string | number },
    category: NotificationCategory,
    languages: string[],
    link?: string,
    prisma: PrismaType,
    titleKey?: TransKey,
    titleVariables?: { [key: string]: string | number },
    userId: string,
}

/**
 * Base function for pushing a notification to a user. If the user 
 * has no devices, emails, or phone numbers to send the message to, 
 * it is still stored in the database for the user to see when they 
 * open the app.
 */
const push = async ({
    bodyKey,
    bodyVariables,
    category,
    languages,
    link,
    prisma,
    titleKey,
    titleVariables,
    userId
}: PushParams) => {
    // Find out which devices can receive this notification, and the daily limit
    const { pushDevices, emails, phoneNumbers, dailyLimit } = await findRecipientsAndLimit(category, prisma, userId);
    // Try connecting to redis
    try {
        const client = await initializeRedis();
        // Increment count in Redis for this user. If it is over the limit, don't send the notification
        const count = await client.incr(`notification:${userId}:${category}`);
        if (dailyLimit && count > dailyLimit) return;
        // Find title and body
        const lng = languages.length > 0 ? languages[0] : 'en';
        const title: string | undefined = titleKey ? i18next.t(`notify:${titleKey}`, { lng, ...(titleVariables ?? {}) }) : undefined;
        const body: string = i18next.t(`notify:${bodyKey}`, { lng, ...(bodyVariables ?? {}) });
        // Send the notification to each device
        const icon = `https://app.vrooli.com/Logo.png`; // TODO location of logo
        for (const device of pushDevices) {
            try {
                const subscription = {
                    endpoint: device.endpoint,
                    keys: {
                        p256dh: device.p256dh,
                        auth: device.auth,
                    },
                }
                const payload = { body, icon, link, title }
                sendPush(subscription, payload);
            } catch (err) {
                logger.error('Error sending push notification', { trace: '0306' });
            }
        }
        // Send the notification to each email (ignore if no title)
        if (emails.length && title) {
            sendMail(emails.map(e => e.emailAddress), title, body);
        }
        // Send the notification to each phone number
        // for (const phoneNumber of phoneNumbers) {
        //     fdasfsd
        // }
        // Store the notification in the database
        await prisma.notification.create({
            data: {
                category,
                // Title or first 20 characters of body
                title: title ?? `${body.substring(0, 10)}...`,
                description: body,
                link,
                imgLink: icon,
                user: { connect: { id: userId } },
            }
        })
    }
    // If Redis fails, let the user through. It's not their fault. 
    catch (error) {
        logger.error('Error occured while connecting or accessing redis server', { trace: '0305', error });
    }
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

type NotifyResultType = {
    toUser: (userId: string) => Promise<void>,
    toOrganization: (organizationId: string) => Promise<void>,
}

/**
 * Class returned by each notify function. Allows us to either
 * send the notification to one user, or to all admins of an organization
 */
const NotifyResult = (prisma: PrismaType, params: Omit<PushParams, 'userId'>): NotifyResultType => ({
    /**
     * Sends a notification to a user
     * @param userId The user's id
     */
    toUser: async (userId) => {
        await push({ ...params, userId, prisma });
    },
    /**
     * Sends a notification to an organization
     * @param organizationId The organization's id
     */
    toOrganization: async (organizationId) => {
        // Find every admin of the organization
        const admins = await prisma.member.findMany({
            where: {
                AND: [
                    { organizationId },
                    { isAdmin: true }
                ]
            },
            select: { userId: true }
        })
        const adminIds = admins ? admins.map(a => a.userId) : [];
        // Send a notification to each admin
        for (const adminId of adminIds) {
            await push({ ...params, userId: adminId, prisma });
        }
    },
})

/**
 * Handles sending and registering notifications for a user or organization. 
 * Organization notifications are sent to every admin of the organization.
 * Notifications settings and devices are queried from the main database.
 * Notification limits are tracked using Redis.
 */
export const Notify = (prisma: PrismaType, languages: string[]) => ({
    /**
     * Sets up a push device to receive notifications
     */
    registerPushDevice: async ({ endpoint, p256dh, auth, expires, userId }: {
        endpoint: string,
        p256dh: string,
        auth: string,
        expires?: Date,
        userId: string,
    }) => {
        // Check if the device is already registered
        const device = await prisma.notification_device.findUnique({
            where: { endpoint },
            select: { id: true }
        })
        // If it is, update the auth and p256dh keys
        if (device) {
            await prisma.notification_device.update({
                where: { id: device.id },
                data: { auth, p256dh, expires }
            })
        }
        // If it isn't, create a new device
        await prisma.notification_device.create({
            data: {
                endpoint,
                auth,
                p256dh,
                expires,
                user: { connect: { id: userId } }
            }
        })
    },
    /**
     * Removes a push device from the database
     * @param deviceId The device's id
     * @param userId The user's id
     */
    unregisterPushDevice: async (deviceId: string, userId: string) => {
        // Check if the device is registered to the user
        const device = await prisma.notification_device.findUnique({
            where: { id: deviceId },
            select: { userId: true }
        })
        if (!device || device.userId !== userId)
            throw new CustomError('0307', 'PushDeviceNotFound', languages);
        // If it is, delete it  
        await prisma.notification_device.delete({ where: { id: deviceId } })
    },
    /**
     * Updates a user's notification settings
     * @param settings The new settings
     */
    updateSettings: async (settings: NotificationSettings, userId: string) => {
        await updateNotificationSettings(settings, prisma, userId);
    },
    pushApiOutOfCredits: (): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'ApiOutOfCreditsBody',
        category: 'AccountCreditsOrApi',
        languages,
        prisma,
        titleKey: 'ApiOutOfCreditsTitle',
    }),
    pushAward: (awardName: string, awardDescription: string): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'AwardEarnedBody',
        bodyVariables: { awardName, awardDescription },
        category: 'Award',
        languages,
        link: '/awards',
        prisma,
        titleKey: 'AwardEarnedTitle',
    }),
    pushIssueActivity: (issueName: string, issueId: string): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'IssueActivityBody',
        bodyVariables: { issueName },
        category: 'IssueActivity',
        languages,
        link: `/issues/${issueId}`,
        prisma,
        titleKey: 'IssueActivityTitle',
    }),
    pushNewDeviceSignIn: (): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'NewDeviceBody',
        category: 'Security',
        languages,
        prisma,
        titleKey: 'NewDeviceTitle',
    }),
    pushNewQuestionOnObject: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'NewQuestionOnObjectBody',
        bodyVariables: { objectName },
        category: 'NewQuestionOrIssue',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'NewQuestionOnObjectTitle',
    }),
    pushNewIssueOnObject: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'NewIssueOnObjectBody',
        bodyVariables: { objectName },
        category: 'NewQuestionOrIssue',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'NewIssueOnObjectTitle',
    }),
    pushObjectReceivedStar: (objectName: string, objectType: string, objectId: string, totalStars: number): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'ObjectReceivedStarBody',
        bodyVariables: { objectName, count: totalStars },
        category: 'ObjectStarVoteFork',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ObjectReceivedStarTitle',
    }),
    pushObjectReceivedUpvote: (objectName: string, objectType: string, objectId: string, totalScore: number): NotifyResultType => NotifyResult(prisma, {
        bodyKey: 'ObjectReceivedUpvoteBody',
        bodyVariables: { objectName, count: totalScore },
        category: 'ObjectStarVoteFork',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ObjectReceivedUpvoteTitle',
    }),
    pushObjectReceivedFork: (objectName: string, objectType: string, objectId: string): NotifyResultType => {
        const title = 'New Fork';
        const body = `${objectName} was forked!`;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: 'ObjectStarVoteFork', link, prisma, title });
    },
    pushReportClosedObjectDeleted: (objectName: string): NotifyResultType => {
        const title = 'Object Deleted';
        const body = `Sorry, the community decided to delete ${objectName}. Your reputation score has been reduced.`;
        return NotifyResult(prisma, { body, category: 'ReportClose', prisma, title });
    },
    pushReportClosedObjectHidden: (objectName: string, objectType: string, objectId: string): NotifyResultType => {
        const title = 'Object Hidden';
        const body = `${objectName} has been hidden from the community. Please fix the issues mentioned in its reports and publish an updated version if you'd like it to be visible again.`;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.ReportClose, link, prisma, title });
    },
    pushReportClosedAccountSuspended: (objectName: string): NotifyResultType => {
        const title = 'Account Suspended';
        const body = `Your account has been suspended due to the reports on ${objectName}. Please contact us if you think this was a mistake.`;
        return NotifyResult(prisma, { body, category: NotificationCategory.ReportClose, prisma, title });
    },
    pushPromotion: (title: string, body: string, link: string): NotifyResultType => {
        return NotifyResult(prisma, { body, category: NotificationCategory.Promotion, link, prisma, title });
    },
    pushPullRequestAccepted: (objectName: string, objectType: string, objectId: string): NotifyResultType => {
        const title = 'Pull Request Accepted';
        const body = `Your improvements for ${objectName} have been accepted! Thank you for your contribution.`;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.PullRequestClose, link, prisma, title });
    },
    pushPullRequestRejected: (objectName: string, objectType: string, objectId: string): NotifyResultType => {
        const title = 'Pull Request Rejected';
        const body = `Sorry, your improvements for ${objectName} have been rejected.`;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.PullRequestClose, link, prisma, title });
    },
    pushQuestionActivity: (questionName: string, questionId: string): NotifyResultType => {
        const title = 'Question Activity';
        const body = `New activity on question "${questionName}"`;
        const link = `/questions/${questionId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.QuestionActivty, link, prisma, title });
    },
    pushRunStartedAutomatically: (runName: string, runId: string): NotifyResultType => {
        const title = 'Run Started';
        const body = `Your run "${runName}" has started! We'll let you know when it's finished.`;
        const link = `/runs/${runId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Run, link, prisma, title });
    },
    pushRunComplete: (runTitle: string, runId: string): NotifyResultType => {
        const title = `Run completed!`;
        const body = `Your run "${runTitle}" is complete! Press here to view the results.`;
        const link = `/runs/${runId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Run, link, prisma, title });
    },
    pushRunFail: (runTitle: string, runId: string): NotifyResultType => {
        const title = `Run failed!`;
        const body = `Your run "${runTitle}" failed. Press here to see more details.`;
        const link = `/runs/${runId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Run, link, prisma, title });
    },
    pushScheduleOrganization: (meetingName: string, startTime: Date): NotifyResultType => {
        const startLabel = getEventStartLabel(startTime);
        const title = 'Meeting reminder';
        const body = `${meetingName} is starting in ${startLabel}!`;
        const link = `/meeting/${meetingName}`; //TODO
        return NotifyResult(prisma, { body, category: NotificationCategory.Schedule, link, prisma, title });
    },
    pushScheduleUser: (title: string, startTime: Date): NotifyResultType => {
        const startLabel = getEventStartLabel(startTime);
        const body = `${title} is starting in ${startLabel}!`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Schedule, prisma });
    },
    pushStreakReminder: (timeToReset: Date): NotifyResultType => {
        const startLabel = getEventStartLabel(timeToReset);
        const title = 'Streak reminder';
        const body = `Your streak will be broken in ${startLabel}. Complete a routine to keep your streak going!`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Streak, prisma, title });
    },
    pushStreakBroken: (): NotifyResultType => {
        const title = 'Streak broken';
        const body = 'Your streak has been broken. Complete a routine to start a new streak!';
        return NotifyResult(prisma, { body, category: NotificationCategory.Streak, prisma, title });
    },
    pushTransferRequest: (transferId: string, objectType: string): NotifyResultType => {
        const title = 'Transfer Request';
        const body = 'Someone wants to send you an object!';
        const link = `/transfers/${transferId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Transfer, link, prisma, title });
    },
    pushTransferAccepted: (objectName: string | null | undefined, objectType: string, objectId: string): NotifyResultType => {
        const title = 'Transfer Accepted';
        const body = `Your transfer request ${objectName ? 'for ' : ''}${objectName}has been accepted!`;;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Transfer, link, prisma, title });
    },
    pushTransferRejected: (objectName: string | null | undefined, objectType: string, objectId: string): NotifyResultType => {
        const title = 'Transfer Rejected';
        const body = `Your transfer request ${objectName ? 'for ' : ''}${objectName}has been rejected.`;;
        const link = `/${objectType}/${objectId}`;
        return NotifyResult(prisma, { body, category: NotificationCategory.Transfer, link, prisma, title });
    },
    pushUserInvite: (friendUsername: string): NotifyResultType => {
        const body = `${friendUsername} has created an account using your invite code! Enjoy your free month of premium😎`;
        return NotifyResult(prisma, { body, category: NotificationCategory.UserInvite, prisma });
    },
});