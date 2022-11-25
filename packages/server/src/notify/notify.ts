import { CustomError, logger, SubscribableObject } from "../events";
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
    silents?: boolean[],
    titleKey?: TransKey,
    titleVariables?: { [key: string]: string | number },
    userIds: string[],
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
    silents,
    titleKey,
    titleVariables,
    userIds
}: PushParams) => {
    // Find out which devices can receive this notification, and the daily limit
    const devicesAndLimits = await findRecipientsAndLimit(category, prisma, userIds);
    // Find title and body
    const lng = languages.length > 0 ? languages[0] : 'en';
    const title: string | undefined = titleKey ? i18next.t(`notify:${titleKey}`, { lng, ...(titleVariables ?? {}) }) : undefined;
    const body: string = i18next.t(`notify:${bodyKey}`, { lng, ...(bodyVariables ?? {}) });
    const icon = `https://app.vrooli.com/Logo.png`; // TODO location of logo
    // Try connecting to redis
    try {
        const client = await initializeRedis();
        // For each user
        for (let i = 0; i < userIds.length; i++) {
            const { pushDevices, emails, phoneNumbers, dailyLimit } = devicesAndLimits[i];
            let currSilent = silents ? silents[i] : false;
            // Increment count in Redis for this user. If it is over the limit, make the notification silent
            const count = await client.incr(`notification:${userIds[i]}:${category}`);
            if (dailyLimit && count > dailyLimit) currSilent = true;
            // Send the notification to each device, if not silent
            if (!currSilent) {
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
            }
        }
        // Store the notifications in the database
        await prisma.notification.createMany({
            data: userIds.map(userId => ({
                category,
                // Title or first 20 characters of body
                title: title ?? `${body.substring(0, 10)}...`,
                description: body,
                link,
                imgLink: icon,
                userId,
            }))
        });
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
    toOrganization: (organizationId: string, excludeUserId?: string) => Promise<void>,
    toSubscribers: (objectType: SubscribableObject, objectId: string, excludeUserId?: string) => Promise<void>,
}

/**
 * Class returned by each notify function. Allows us to either
 * send the notification to one user, or to all admins of an organization
 */
const NotifyResult = (params: Omit<PushParams, 'userIds'>): NotifyResultType => ({
    /**
     * Sends a notification to a user
     * @param userId The user's id
     */
    toUser: async (userId) => {
        await push({ ...params, userIds: [userId] });
    },
    /**
     * Sends a notification to an organization
     * @param organizationId The organization's id
     * @param excludeUserId The user to exclude from the notification 
     * (usually the user who triggered the notification)
     */
    toOrganization: async (organizationId, excludeUserId) => {
        // Find every admin of the organization, excluding the user who triggered the notification
        const admins = await params.prisma.member.findMany({
            where: {
                AND: [
                    { organizationId },
                    { isAdmin: true },
                    { userId: { not: excludeUserId } }
                ]
            },
            select: { userId: true }
        })
        const adminIds = admins ? admins.map(a => a.userId) : [];
        // Send a notification to each admin
        await push({ ...params, userIds: adminIds });
    },
    /**
     * Sends a notification to all subscribers of an object
     * @param objectType The type of object
     * @param objectId The object's id
     * @param excludeUserId The user to exclude from the notification
     */
    toSubscribers: async (objectType, objectId, excludeUserId) => {
        // Find all subscribers of the object. There may be a lot, 
        // so we need to do this in batches
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await params.prisma.notification_subscription.findMany({
                where: {
                    AND: [
                        { objectType },
                        { objectId },
                        { userId: { not: excludeUserId } }
                    ]
                },
                select: { userId: true, silent: true },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            // Send a notification to each subscriber
            await push({ ...params, userIds: batch.map(b => b.userId), silents: batch.map(b => b.silent) });
        } while (currentBatchSize === batchSize);
    }
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
    pushApiOutOfCredits: (): NotifyResultType => NotifyResult({
        bodyKey: 'ApiOutOfCreditsBody',
        category: 'AccountCreditsOrApi',
        languages,
        prisma,
        titleKey: 'ApiOutOfCreditsTitle',
    }),
    pushAward: (awardName: string, awardDescription: string): NotifyResultType => NotifyResult({
        bodyKey: 'AwardEarnedBody',
        bodyVariables: { awardName, awardDescription },
        category: 'Award',
        languages,
        link: '/awards',
        prisma,
        titleKey: 'AwardEarnedTitle',
    }),
    pushIssueActivity: (issueName: string, issueId: string): NotifyResultType => NotifyResult({
        bodyKey: 'IssueActivityBody',
        bodyVariables: { issueName },
        category: 'IssueActivity',
        languages,
        link: `/issues/${issueId}`,
        prisma,
        titleKey: 'IssueActivityTitle',
    }),
    pushNewDeviceSignIn: (): NotifyResultType => NotifyResult({
        bodyKey: 'NewDeviceBody',
        category: 'Security',
        languages,
        prisma,
        titleKey: 'NewDeviceTitle',
    }),
    pushNewEmailVerification: (): NotifyResultType => NotifyResult({
        bodyKey: 'NewEmailVerificationBody',
        category: 'Security',
        languages,
        prisma,
        titleKey: 'NewEmailVerificationTitle',
    }),
    pushNewQuestionOnObject: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'NewQuestionOnObjectBody',
        bodyVariables: { objectName },
        category: 'NewQuestionOrIssue',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'NewQuestionOnObjectTitle',
    }),
    pushNewIssueOnObject: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'NewIssueOnObjectBody',
        bodyVariables: { objectName },
        category: 'NewQuestionOrIssue',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'NewIssueOnObjectTitle',
    }),
    pushObjectReceivedStar: (objectName: string, objectType: string, objectId: string, totalStars: number): NotifyResultType => NotifyResult({
        bodyKey: 'ObjectReceivedStarBody',
        bodyVariables: { objectName, count: totalStars },
        category: 'ObjectStarVoteFork',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ObjectReceivedStarTitle',
    }),
    pushObjectReceivedUpvote: (objectName: string, objectType: string, objectId: string, totalScore: number): NotifyResultType => NotifyResult({
        bodyKey: 'ObjectReceivedUpvoteBody',
        bodyVariables: { objectName, count: totalScore },
        category: 'ObjectStarVoteFork',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ObjectReceivedUpvoteTitle',
    }),
    pushObjectReceivedFork: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'ObjectReceivedForkBody',
        bodyVariables: { objectName },
        category: 'ObjectStarVoteFork',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ObjectReceivedForkTitle',
    }),
    pushReportClosedObjectDeleted: (objectName: string): NotifyResultType => NotifyResult({
        bodyKey: 'ReportClosedObjectDeletedBody',
        bodyVariables: { objectName },
        category: 'ReportClose',
        languages,
        prisma,
        titleKey: 'ReportClosedObjectDeletedTitle',
    }),
    pushReportClosedObjectHidden: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'ReportClosedObjectHiddenBody',
        bodyVariables: { objectName },
        category: 'ReportClose',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'ReportClosedObjectHiddenTitle',
    }),
    pushReportClosedAccountSuspended: (objectName: string): NotifyResultType => NotifyResult({
        bodyKey: 'ReportClosedAccountSuspendedBody',
        bodyVariables: { objectName },
        category: 'ReportClose',
        languages,
        prisma,
        titleKey: 'ReportClosedAccountSuspendedTitle',
    }),
    pushPullRequestAccepted: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'PullRequestAcceptedBody',
        bodyVariables: { objectName },
        category: 'PullRequestClose',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'PullRequestAcceptedTitle',
    }),
    pushPullRequestRejected: (objectName: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: 'PullRequestRejectedBody',
        bodyVariables: { objectName },
        category: 'PullRequestClose',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'PullRequestRejectedTitle',
    }),
    pushQuestionActivity: (questionName: string, questionId: string): NotifyResultType => NotifyResult({
        bodyKey: 'QuestionActivityBody',
        bodyVariables: { questionName },
        category: 'QuestionActivity',
        languages,
        link: `/questions/${questionId}`,
        prisma,
        titleKey: 'QuestionActivityTitle',
    }),
    pushRunStartedAutomatically: (runName: string, runId: string): NotifyResultType => NotifyResult({
        bodyKey: 'RunStartedAutomaticallyBody',
        bodyVariables: { runName },
        category: 'Run',
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: 'RunStartedAutomaticallyTitle',
    }),
    pushRunCompletedAutomatically: (runName: string, runId: string): NotifyResultType => NotifyResult({
        bodyKey: 'RunCompletedAutomaticallyBody',
        bodyVariables: { runName },
        category: 'Run',
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: 'RunCompletedAutomaticallyTitle',
    }),
    pushRunFailedAutomatically: (runName: string, runId: string): NotifyResultType => NotifyResult({
        bodyKey: 'RunFailedAutomaticallyBody',
        bodyVariables: { runName },
        category: 'Run',
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: 'RunFailedAutomaticallyTitle',
    }),
    pushScheduleOrganization: (meetingName: string, meetingId: string, startTime: Date): NotifyResultType => NotifyResult({
        bodyKey: 'ScheduleOrganizationBody',
        bodyVariables: { meetingName, startLabel: getEventStartLabel(startTime) },
        category: 'Schedule',
        languages,
        link: `/meeting/${meetingId}`,
        prisma,
        titleKey: 'ScheduleOrganizationTitle',
    }),
    pushScheduleUser: (title: string, startTime: Date): NotifyResultType => NotifyResult({
        bodyKey: 'ScheduleUserBody',
        bodyVariables: { title, startLabel: getEventStartLabel(startTime) },
        category: 'Schedule',
        languages,
        prisma,
    }),
    pushStreakReminder: (timeToReset: Date): NotifyResultType => NotifyResult({
        bodyKey: 'StreakReminderBody',
        bodyVariables: { endLabel: getEventStartLabel(timeToReset) },
        category: 'Streak',
        languages,
        prisma,
    }),
    pushStreakBroken: (): NotifyResultType => NotifyResult({
        bodyKey: 'StreakBrokenBody',
        category: 'Streak',
        languages,
        prisma,
        titleKey: 'StreakBrokenTitle',
    }),
    pushTransferRequest: (transferId: string, objectType: string): NotifyResultType => NotifyResult({
        bodyKey: 'TransferRequestBody',
        category: 'Transfer',
        languages,
        link: `/transfers/${transferId}`,
        prisma,
        titleKey: 'TransferRequestTitle',
    }),
    pushTransferAccepted: (objectName: string | null | undefined, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: objectName ? 'TransferAcceptedWithNameBody' : 'TransferAcceptedTitle',
        bodyVariables: objectName ? { objectName } : {},
        category: 'Transfer',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'TransferAcceptedTitle',
    }),
    pushTransferRejected: (objectName: string | null | undefined, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: objectName ? 'TransferRejectedWithNameBody' : 'TransferRejectedTitle',
        bodyVariables: objectName ? { objectName } : {},
        category: 'Transfer',
        languages,
        link: `/${objectType}/${objectId}`,
        prisma,
        titleKey: 'TransferRejectedTitle',
    }),
    pushUserInvite: (friendUsername: string): NotifyResultType => NotifyResult({
        bodyKey: 'UserInviteBody',
        bodyVariables: { friendUsername },
        category: 'UserInvite',
        languages,
        prisma,
    }),
});