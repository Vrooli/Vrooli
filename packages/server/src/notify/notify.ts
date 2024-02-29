import { GqlModelType, IssueStatus, LINKS, NotificationSettingsUpdateInput, PullRequestStatus, PushDevice, ReportStatus, SubscribableObject, Success } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next, { TFuncKey } from "i18next";
import { selectHelper } from "../builders/selectHelper";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { GraphQLInfo, PartialGraphQLInfo } from "../builders/types";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { subscribableMapper } from "../events/subscriber";
import { ModelMap } from "../models/base";
import { PushDeviceModel } from "../models/base/pushDevice";
import { OrganizationModelLogic } from "../models/base/types";
import { withRedis } from "../redisConn";
import { sendMail } from "../tasks/email/queue";
import { sendPush } from "../tasks/push/queue";
import { PrismaType, SessionUserToken } from "../types";
import { batch } from "../utils/batch";
import { findRecipientsAndLimit, updateNotificationSettings } from "./notificationSettings";

export type NotificationUrgency = "low" | "normal" | "critical";

export type NotificationCategory = "AccountCreditsOrApi" |
    "Award" |
    "IssueStatus" |
    "Message" |
    "NewObjectInOrganization" |
    "NewObjectInProject" |
    "NewQuestionOrIssue" |
    "ObjectActivity" |
    "Promotion" |
    "PullRequestStatus" |
    "QuestionActivity" |
    "ReportStatus" |
    "Run" |
    "Schedule" |
    "Security" |
    "Streak" |
    "Transfer" |
    "UserInvite";

type TransKey = TFuncKey<"notify", undefined>

type PushToUser = {
    delays?: number[],
    bodyVariables?: { [key: string]: string | number }, // Variables can change depending on recipient's language
    languages: string[],
    silent?: boolean,
    titleVariables?: { [key: string]: string | number }, // Variables can change depending on recipient's language
    userId: string,
}

type PushParams = {
    bodyKey?: TransKey,
    category: NotificationCategory,
    link?: string,
    prisma: PrismaType,
    titleKey?: TransKey,
    users: PushToUser[]
}

type NotifyResultParams = {
    bodyKey?: TransKey,
    bodyVariables?: { [key: string]: string | number },
    category: NotificationCategory,
    languages: string[],
    link?: string,
    prisma: PrismaType,
    silent?: boolean,
    titleKey?: TransKey,
    titleVariables?: { [key: string]: string | number },
}

/**
 * Checks if an object type can be subscribed to
 * @param objectType The object type to check
 * @returns True if the object type can be subscribed to
 */
export const isObjectSubscribable = <T extends keyof typeof GqlModelType>(objectType: T): boolean => {
    return objectType in SubscribableObject;
};

/**
 * Base function for pushing a notification to a list of users. If a user 
 * has no devices, emails, or phone numbers to send the message to, 
 * it is still stored in the database for the user to see when they 
 * open the app.
 */
const push = async ({
    bodyKey,
    category,
    link,
    prisma,
    titleKey,
    users,
}: PushParams) => {
    // Find out which devices can receive this notification, and the daily limit
    const devicesAndLimits = await findRecipientsAndLimit(category, prisma, users.map(u => u.userId));
    // Find title and body for each user
    const userTitles: { [userId: string]: string } = {};
    const userBodies: { [userId: string]: string } = {};
    for (const user of users) {
        const lng = user.languages.length > 0 ? user.languages[0] : "en";
        const title: string | undefined = titleKey ? i18next.t(`notify:${titleKey}`, { lng, ...(user.titleVariables ?? {}) }) : undefined;
        const body: string | undefined = bodyKey ? i18next.t(`notify:${bodyKey}`, { lng, ...(user.bodyVariables ?? {}) }) : undefined;
        // At least one of title or body must be defined
        if (!title && !body) throw new CustomError("0362", "InternalError", user.languages);
        userTitles[user.userId] = title ?? `${body!.substring(0, 10)}...`; // If no title, use shortened body
        userBodies[user.userId] = body ?? title!;
    }
    const icon = "https://vrooli.com/Logo.png"; // TODO location of logo
    await withRedis({
        process: async (redisClient) => {
            // For each user
            for (let i = 0; i < users.length; i++) {
                // For each delay
                for (const delay of users[i].delays ?? [0]) {
                    const { pushDevices, emails, phoneNumbers, dailyLimit } = devicesAndLimits[i];
                    let currSilent = users[i].silent ?? false;
                    const currTitle = userTitles[users[i].userId];
                    const currBody = userBodies[users[i].userId];
                    // Increment count in Redis for this user. If it is over the limit, make the notification silent
                    const count = await redisClient.incr(`notification:${users[i].userId}:${category}`);
                    if (dailyLimit && count > dailyLimit) currSilent = true;
                    // Send the notification if not silent
                    if (!currSilent) {
                        const payload = { body: currBody, icon, link, title: currTitle };
                        // Send through socket to display on opened app TODO
                        // emitSocketEvent("notification", fdasfsf, payload);
                        // Send to each push device
                        for (const device of pushDevices) {
                            try {
                                const subscription = {
                                    endpoint: device.endpoint,
                                    keys: {
                                        p256dh: device.p256dh,
                                        auth: device.auth,
                                    },
                                };
                                sendPush(subscription, payload, delay);
                            } catch (err) {
                                logger.error("Error sending push notification", { trace: "0306" });
                            }
                        }
                        // Send to each email (ignore if no title)
                        if (emails.length && currTitle) {
                            sendMail(emails.map(e => e.emailAddress), currTitle, currBody, "", delay);
                        }
                        // Send to each phone number
                        // for (const phoneNumber of phoneNumbers) {
                        //     fdasfsd
                        // }
                    }
                }
            }
            // Store the notifications in the database
            await prisma.notification.createMany({
                data: users.map(({ userId }) => ({
                    category,
                    title: userTitles[userId],
                    description: userBodies[userId],
                    link,
                    imgLink: icon,
                    userId,
                })),
            });
        },
        trace: "0512",
    });
};

/**
 * Creates an appropriate label for a near-future event. 
 * Examples: (5 minutes, 2 hours, 1 day)
 * @param date The date of the event. If it's in the past, we return "now"
 */
const getEventStartLabel = (date: Date) => {
    const now = new Date();
    const diff = date.getTime() - now.getTime();
    if (diff < 0) return "now";
    if (diff < 1000 * 60) return "in a few seconds";
    if (diff < 1000 * 60 * 60) return `in ${Math.round(diff / (1000 * 60))} minutes`;
    if (diff < 1000 * 60 * 60 * 24) return `in ${Math.round(diff / (1000 * 60 * 60))} hours`;
    return `in ${Math.round(diff / (1000 * 60 * 60 * 24))} days`;
};

type NotifyResultType = {
    toUser: (userId: string) => Promise<void>,
    toUsers: (userIds: (string | { userId: string, delays: number[] })[]) => Promise<void>,
    toOrganization: (organizationId: string, excludedUsers?: string[] | string) => Promise<void>,
    toOwner: (owner: { __typename: "User" | "Organization", id: string }, excludedUsers?: string[] | string) => Promise<void>,
    toSubscribers: (objectType: SubscribableObject | `${SubscribableObject}`, objectId: string, excludedUsers?: string[] | string) => Promise<void>,
    toChatParticipants: (chatId: string, excludedUsers?: string[] | string) => Promise<void>,
}

/**
 * Replaces the label placeholders in a title and body's variables with the appropriate, translated labels.
 * @param bodyVariables The variables for the body
 * @param titleVariables The variables for the title
 * @param silent Whether the notification is silent
 * @param prisma Prisma client
 * @param languages Preferred languages for error messages
 * @param users List of userIds and their languages
 * @returns PushToUser list with the variables replaced
 */
const replaceLabels = async (
    bodyVariables: { [key: string]: string | number } | undefined,
    titleVariables: { [key: string]: string | number } | undefined,
    silent: boolean | undefined,
    prisma: PrismaType,
    languages: string[],
    users: Pick<PushToUser, "languages" | "userId" | "delays">[],
): Promise<PushToUser[]> => {
    const labelRegex = /<Label\|([A-z]+):([0-9-]+)>/;
    // Initialize the result
    const result: PushToUser[] = users.map(u => ({ ...u, bodyVariables, titleVariables, silent }));
    // If titleVariables or bodyVariables contains "<Label|objectType:${id}>", we must inject 
    // the object's translated label
    let labelTranslations: { [key: string]: string } = {};
    // Helper function to query for label translations
    const findTranslations = async (objectType: `${GqlModelType}`, objectId: string) => {
        // Ignore if already translated
        if (Object.keys(labelTranslations).length > 0) return;
        const { delegate, display } = ModelMap.getLogic(["delegate", "display"], objectType, true, "replaceLabels 1");
        const labels = await delegate(prisma).findUnique({
            where: { id: objectId },
            select: display().label.select(),
        });
        labelTranslations = labels ?? {};
    };
    // If any object in titleVariables contains <Label|objectType:id>
    if (titleVariables) {
        // Loop through each key
        for (const key of Object.keys(titleVariables)) {
            // If the value is a string
            if (typeof titleVariables[key] === "string") {
                // If the value contains <Label|objectType:id>
                const match = (titleVariables[key] as string).match(labelRegex);
                if (match) {
                    // Find label translations
                    await findTranslations(match[1] as GqlModelType, match[2]);
                    // In each params, replace the matching substring with the label
                    const { display } = ModelMap.getLogic(["display"], match[1] as GqlModelType, true, "replaceLabels 2");
                    for (let i = 0; i < result.length; i++) {
                        result[i][key] = (result[i][key] as string).replace(match[0], display().label.get(labelTranslations, result[i].languages));
                    }
                }
            }
        }
    }
    // If any object in bodyVariables contains <Label|objectType:id>
    if (bodyVariables) {
        // Loop through each key
        for (const key of Object.keys(bodyVariables)) {
            // If the value is a string
            if (typeof bodyVariables[key] === "string") {
                // If the value contains <Label|objectType:id>
                const match = (bodyVariables[key] as string).match(labelRegex);
                if (match) {
                    // Find label translations
                    await findTranslations(match[1] as GqlModelType, match[2]);
                    // In each params, replace the matching substring with the label
                    const { display } = ModelMap.getLogic(["display"], match[1] as GqlModelType, true, "replaceLabels 3");
                    for (let i = 0; i < result.length; i++) {
                        result[i][key] = (result[i][key] as string).replace(match[0], display().label.get(labelTranslations, result[i].languages));
                    }
                }
            }
        }
    }
    return result;
};

/**
 * Class returned by each notify function. Allows us to either
 * send the notification to one user, or to all admins of an organization
 */
const NotifyResult = ({
    bodyKey,
    bodyVariables,
    category,
    languages,
    link,
    prisma,
    silent,
    titleKey,
    titleVariables,
}: NotifyResultParams): NotifyResultType => ({
    /**
     * Sends a notification to a user
     * @param userId The user's id
     */
    toUser: async (userId) => {
        // Shape and translate the notification for the user
        const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, [{ languages, userId }]);
        // Send the notification
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    /**
     * Sends a notification to multiple users
     * @param userIds The users' ids
     */
    toUsers: async (userIds) => {
        // Shape and translate the notification for each user
        const users = await replaceLabels(
            bodyVariables,
            titleVariables,
            silent,
            prisma,
            languages,
            userIds.map(data => ({
                languages,
                userId: typeof data === "string" ? data : data.userId,
                delays: typeof data === "string" ? undefined : data.delays,
            })));
        // Send the notification to each user
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    /**
     * Sends a notification to an organization
     * @param organizationId The organization's id
     * @param excludedUsers IDs of users to exclude from the notification
     * (usually the user who triggered the notification)
     */
    toOrganization: async (organizationId, excludedUsers) => {
        // Find every admin of the organization, excluding the user who triggered the notification
        const adminData = await ModelMap.get<OrganizationModelLogic>("Organization").query.findAdminInfo(prisma, organizationId, excludedUsers);
        // Shape and translate the notification for each admin
        const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, adminData.map(({ id, languages }) => ({
            languages,
            userId: id,
        })));
        // Send the notification to each admin
        await push({ bodyKey, category, link, prisma, titleKey, users });
    },
    /**
     * Sends a notification to an owner of an object
     * @param owner The owner's id and __typename
     * @param excludedUsers IDs of users to exclude from the notification
     */
    toOwner: async (owner, excludedUsers) => {
        if (owner.__typename === "User") {
            await NotifyResult({ bodyKey, bodyVariables, category, languages, link, prisma, silent, titleKey, titleVariables }).toUser(owner.id);
        } else if (owner.__typename === "Organization") {
            await NotifyResult({ bodyKey, bodyVariables, category, languages, link, prisma, silent, titleKey, titleVariables }).toOrganization(owner.id, excludedUsers);
        }
    },
    /**
     * Sends a notification to all subscribers of an object
     * @param objectType The __typename of object
     * @param objectId The object's id
     * @param excludedUsers IDs of users to exclude from the notification
     */
    toSubscribers: async (objectType, objectId, excludedUsers) => {
        await batch<Prisma.notification_subscriptionFindManyArgs>({
            objectType: "NotificationSubscription",
            processBatch: async (batch: { subscriberId: string, silent: boolean }[], prisma) => {
                // Shape and translate the notification for each subscriber
                const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, batch.map(({ subscriberId, silent }) => ({
                    languages,
                    silent,
                    userId: subscriberId,
                })));
                // Send the notification to each subscriber
                await push({ bodyKey, category, link, prisma, titleKey, users });
            },
            select: { subscriberId: true, silent: true },
            trace: "0467",
            where: {
                AND: [
                    { [subscribableMapper[objectType]]: { id: objectId } },
                    ...(typeof excludedUsers === "string" ?
                        [{ subscriberId: { not: excludedUsers } }] :
                        Array.isArray(excludedUsers) ?
                            [{ subscriberId: { notIn: excludedUsers } }] :
                            []
                    ),
                ],
            },
        });
    },
    /**
     * Sends a notification to all participants of a chat
     * @param chatId The chat's id
     * @param excludedUsers IDs of users to exclude from the notification
     */
    toChatParticipants: async (chatId, excludedUsers) => {
        await batch<Prisma.chat_participantsFindManyArgs>({
            objectType: "ChatParticipant",
            processBatch: async (batch: { user: { id: string } }[], prisma) => {
                // Shape and translate the notification for each participant
                const users = await replaceLabels(bodyVariables, titleVariables, silent, prisma, languages, batch.map(({ user }) => ({
                    languages: ["en"], //TODO need to store user languages in db , then can update this
                    userId: user.id,
                })));
                // Send the notification to each participant
                await push({ bodyKey, category, link, prisma, titleKey, users });
            },
            select: { user: { select: { id: true } } },
            trace: "0498",
            where: typeof excludedUsers === "string"
                ? { AND: [{ chatId }, { userId: { not: excludedUsers } }] }
                : Array.isArray(excludedUsers)
                    ? { AND: [{ chatId }, { userId: { notIn: excludedUsers } }] }
                    : { chatId },
        });
    },
});

/**
 * Handles sending and registering notifications for a user or organization. 
 * Organization notifications are sent to every admin of the organization.
 * Notifications settings and devices are queried from the main database.
 * Notification limits are tracked using Redis.
 */
export const Notify = (prisma: PrismaType, languages: string[]) => ({
    /** Sets up a push device to receive notifications */
    registerPushDevice: async ({ endpoint, p256dh, auth, expires, userData, info }: {
        endpoint: string,
        p256dh: string,
        auth: string,
        expires?: Date,
        name?: string,
        userData: SessionUserToken,
        info: GraphQLInfo | PartialGraphQLInfo,
    }): Promise<PushDevice> => {
        const partialInfo = toPartialGqlInfo(info, PushDeviceModel.format.gqlRelMap, userData.languages, true);
        let select: { [key: string]: any } | undefined;
        let result: any = {};
        try {
            select = selectHelper(partialInfo)?.select;
            // Check if the device is already registered
            const device = await prisma.push_device.findUnique({
                where: { endpoint },
                select: { id: true },
            });
            // If it is, update the auth and p256dh keys
            if (device) {
                result = await prisma.push_device.update({
                    where: { id: device.id },
                    data: { auth, p256dh, expires },
                    select,
                });
            }
            // If it isn't, create a new device
            else {
                result = await prisma.push_device.create({
                    data: {
                        endpoint,
                        auth,
                        p256dh,
                        expires,
                        user: { connect: { id: userData.id } },
                    },
                    select,
                });
            }
            return result;
        } catch (error) {
            throw new CustomError("0452", "InternalError", userData.languages, { error, select, result });
        }
    },
    /**
     * Removes a push device from the database
     * @param deviceId The device's id
     * @param userId The user's id
     */
    unregisterPushDevice: async (deviceId: string, userId: string): Promise<Success> => {
        // Check if the device is registered to the user
        const device = await prisma.push_device.findUnique({
            where: { id: deviceId },
            select: { userId: true },
        });
        if (!device || device.userId !== userId)
            throw new CustomError("0307", "PushDeviceNotFound", languages);
        // If it is, delete it  
        const deletedDevice = await prisma.push_device.delete({ where: { id: deviceId } });
        return { __typename: "Success" as const, success: Boolean(deletedDevice) };
    },
    /**
     * Updates a user's notification settings
     * @param settings The new settings
     */
    updateSettings: async (settings: NotificationSettingsUpdateInput, userId: string) => {
        await updateNotificationSettings(settings, prisma, userId);
    },
    pushApiOutOfCredits: (): NotifyResultType => NotifyResult({
        bodyKey: "ApiOutOfCreditsBody",
        category: "AccountCreditsOrApi",
        languages,
        prisma,
        titleKey: "ApiOutOfCreditsTitle",
    }),
    pushAward: (awardName: string, awardDescription: string): NotifyResultType => NotifyResult({
        bodyKey: "AwardEarnedBody",
        bodyVariables: { awardName, awardDescription },
        category: "Award",
        languages,
        link: "/awards",
        prisma,
        titleKey: "AwardEarnedTitle",
    }),
    pushIssueStatusChange: (
        issueId: string,
        objectId: string,
        objectType: GqlModelType | `${GqlModelType}`,
        status: IssueStatus | `${IssueStatus}`,
    ): NotifyResultType => NotifyResult({
        bodyKey: `IssueStatus${status}Body`,
        bodyVariables: { issueName: `<Label|Issue:${issueId}>`, objectName: `<Label|${objectType}:${objectId}>` },
        category: "IssueStatus",
        languages,
        link: `/issues/${issueId}`,
        prisma,
        titleKey: `IssueStatus${status}Title`,
    }),
    pushMessageReceived: (messageId: string, senderId: string): NotifyResultType => NotifyResult({
        bodyKey: "MessageReceivedBody",
        bodyVariables: { senderName: `<Label|User:${senderId}>` },
        category: "Message",
        languages,
        link: `/messages/${messageId}`,
        prisma,
        titleKey: "MessageReceivedTitle",
    }),
    pushNewDeviceSignIn: (): NotifyResultType => NotifyResult({
        bodyKey: "NewDeviceBody",
        category: "Security",
        languages,
        prisma,
        titleKey: "NewDeviceTitle",
    }),
    pushNewEmailVerification: (): NotifyResultType => NotifyResult({
        bodyKey: "NewEmailVerificationBody",
        category: "Security",
        languages,
        prisma,
        titleKey: "NewEmailVerificationTitle",
    }),
    pushNewQuestionOnObject: (objectType: `${GqlModelType}`, objectId: string, questionId: string): NotifyResultType => NotifyResult({
        bodyKey: "NewQuestionOnObjectBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "NewQuestionOrIssue",
        languages,
        link: `/questions/${questionId}`,
        prisma,
        titleKey: "NewQuestionOnObjectTitle",
    }),
    pushNewObjectInOrganization: (objectType: `${GqlModelType}`, objectId: string, organizationId: string): NotifyResultType => NotifyResult({
        bodyKey: "NewObjectInOrganizationBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, organizationName: `<Label|Organization:${organizationId}>` },
        category: "NewObjectInOrganization",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "NewObjectInOrganizationTitle",
        titleVariables: { organizationName: `<Label|Organization:${organizationId}>` },
    }),
    pushNewObjectInProject: (objectType: `${GqlModelType}`, objectId: string, projectId: string): NotifyResultType => NotifyResult({
        bodyKey: "NewObjectInProjectBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, projectName: `<Label|Project:${projectId}>` },
        category: "NewObjectInProject",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "NewObjectInProjectTitle",
        titleVariables: { projectName: `<Label|Project:${projectId}>` },
    }),
    pushObjectReceivedBookmark: (objectType: `${GqlModelType}`, objectId: string, totalBookmarks: number): NotifyResultType => NotifyResult({
        bodyKey: "ObjectReceivedBookmarkBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalBookmarks },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedBookmarkTitle",
    }),
    pushObjectReceivedComment: (objectType: `${GqlModelType}`, objectId: string, totalComments: number): NotifyResultType => NotifyResult({
        bodyKey: "ObjectReceivedCommentBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalComments },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}?comments`,
        prisma,
        titleKey: "ObjectReceivedCommentTitle",
    }),
    pushObjectReceivedCopy: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedCopyTitle",
        titleVariables: { objectName: `<Label|${objectType}:${objectId}>` },
    }),
    pushObjectReceivedUpvote: (objectType: `${GqlModelType}`, objectId: string, totalScore: number): NotifyResultType => NotifyResult({
        bodyKey: "ObjectReceivedUpvoteBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalScore },
        category: "ObjectActivity",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "ObjectReceivedUpvoteTitle",
    }),
    /**
     * NOTE: If object is being deleted, this should be called before the object is deleted. 
     * Otherwise, we cannot display the name of the object in the notification.
     */
    pushReportStatusChange: (
        reportId: string,
        objectId: string,
        objectType: GqlModelType | `${GqlModelType}`,
        status: ReportStatus | `${ReportStatus}`,
    ): NotifyResultType => NotifyResult({
        bodyKey: `ReportStatus${status}Body`,
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "ReportStatus",
        languages,
        link: `/${LINKS[objectType]}/${objectId}/reports/${reportId}`,
        prisma,
        titleKey: `ReportStatus${status}Title`,
    }),
    pushPullRequestStatusChange: (
        reportId: string,
        objectId: string,
        objectType: GqlModelType | `${GqlModelType}`,
        status: PullRequestStatus | `${PullRequestStatus}`,
    ): NotifyResultType => NotifyResult({
        bodyKey: `PullRequestStatus${status}Body`,
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "PullRequestStatus",
        languages,
        link: `/${LINKS[objectType]}/${objectId}/pulls/${reportId}`,
        prisma,
        titleKey: `PullRequestStatus${status}Title`,
    }),
    pushQuestionActivity: (questionId: string): NotifyResultType => NotifyResult({
        bodyKey: "QuestionActivityBody",
        bodyVariables: { objectName: `<Label|Question:${questionId}>` },
        category: "QuestionActivity",
        languages,
        link: `/questions/${questionId}`,
        prisma,
        titleKey: "QuestionActivityTitle",
    }),
    pushRunStartedAutomatically: (runId: string): NotifyResultType => NotifyResult({
        bodyKey: "RunStartedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunStartedAutomaticallyTitle",
    }),
    pushRunCompletedAutomatically: (runId: string): NotifyResultType => NotifyResult({
        bodyKey: "RunCompletedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunCompletedAutomaticallyTitle",
    }),
    pushRunFailedAutomatically: (runId: string): NotifyResultType => NotifyResult({
        bodyKey: "RunFailedAutomaticallyBody",
        bodyVariables: { runName: `<Label|RunRoutine:${runId}>` },
        category: "Run",
        languages,
        link: `/runs/${runId}`,
        prisma,
        titleKey: "RunFailedAutomaticallyTitle",
    }),
    pushScheduleReminder: (scheduleForId: string, scheduleForType: GqlModelType, startTime: Date): NotifyResultType => NotifyResult({
        bodyKey: "ScheduleUserBody",
        bodyVariables: { title: `<Label|${scheduleForType}:${scheduleForId}>`, startLabel: getEventStartLabel(startTime) },
        category: "Schedule",
        languages,
        link: scheduleForType === "Meeting" ? `/meeting/${scheduleForId}` : undefined,
        prisma,
    }),
    pushStreakReminder: (timeToReset: Date): NotifyResultType => NotifyResult({
        bodyKey: "StreakReminderBody",
        bodyVariables: { endLabel: getEventStartLabel(timeToReset) },
        category: "Streak",
        languages,
        prisma,
    }),
    pushStreakBroken: (): NotifyResultType => NotifyResult({
        bodyKey: "StreakBrokenBody",
        category: "Streak",
        languages,
        prisma,
        titleKey: "StreakBrokenTitle",
    }),
    pushTransferRequestSend: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: "TransferRequestSendBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/transfers/${transferId}`,
        prisma,
        titleKey: "TransferRequestSendTitle",
    }),
    pushTransferRequestReceive: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: "TransferRequestReceiveBody",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/transfers/${transferId}`,
        prisma,
        titleKey: "TransferRequestReceiveTitle",
    }),
    pushTransferAccepted: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: "TransferAcceptedTitle",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "TransferAcceptedTitle",
    }),
    pushTransferRejected: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
        bodyKey: "TransferRejectedTitle",
        bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        category: "Transfer",
        languages,
        link: `/${LINKS[objectType]}/${objectId}`,
        prisma,
        titleKey: "TransferRejectedTitle",
    }),
    pushUserInvite: (friendUsername: string): NotifyResultType => NotifyResult({
        bodyKey: "UserInviteBody",
        bodyVariables: { friendUsername },
        category: "UserInvite",
        languages,
        prisma,
    }),
});
