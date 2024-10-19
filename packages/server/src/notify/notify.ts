import { DAYS_1_MS, GqlModelType, HOURS_1_MS, IssueStatus, LINKS, MINUTES_1_MS, NotificationSettingsUpdateInput, PullRequestStatus, PushDevice, ReportStatus, SubscribableObject, Success } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next, { TFuncKey } from "i18next";
import { selectHelper } from "../builders/selectHelper";
import { toPartialGqlInfo } from "../builders/toPartialGqlInfo";
import { GraphQLInfo, PartialGraphQLInfo, PrismaDelegate } from "../builders/types";
import { prismaInstance } from "../db/instance";
import { CustomError } from "../events/error";
import { logger } from "../events/logger";
import { subscribableMapper } from "../events/subscriber";
import { ModelMap } from "../models/base";
import { PushDeviceModel } from "../models/base/pushDevice";
import { TeamModelLogic } from "../models/base/types";
import { withRedis } from "../redisConn";
import { sendMail } from "../tasks/email/queue";
import { sendPush } from "../tasks/push/queue";
import { SessionUserToken } from "../types";
import { batch } from "../utils/batch";
import { findRecipientsAndLimit, updateNotificationSettings } from "./notificationSettings";

const APP_ICON = "https://vrooli.com/apple-touch-icon.webp";

export type NotificationUrgency = "low" | "normal" | "critical";

export type NotificationCategory =
    "AccountCreditsOrApi" |
    "Award" |
    "IssueStatus" |
    "Message" |
    "NewObjectInTeam" |
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
    titleKey?: TransKey,
    users: PushToUser[]
}

type NotifyResultParams = {
    bodyKey?: TransKey,
    bodyVariables?: { [key: string]: string | number },
    category: NotificationCategory,
    languages: string[],
    link?: string,
    silent?: boolean,
    titleKey?: TransKey,
    titleVariables?: { [key: string]: string | number },
}

/**
 * Checks if an object type can be subscribed to
 * @param objectType The object type to check
 * @returns True if the object type can be subscribed to
 */
export function isObjectSubscribable<T extends keyof typeof GqlModelType>(objectType: T): boolean {
    return objectType in SubscribableObject;
}

/**
 * Base function for pushing a notification to a list of users. If a user 
 * has no devices, emails, or phone numbers to send the message to, 
 * it is still stored in the database for the user to see when they 
 * open the app.
 */
async function push({
    bodyKey,
    category,
    link,
    titleKey,
    users,
}: PushParams) {
    // Find out which devices can receive this notification, and the daily limit
    const devicesAndLimits = await findRecipientsAndLimit(category, users.map(u => u.userId));
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
                    // Increment count in Redis for this user. If it is over the limit, make the notification silent. 
                    // If redis isn't available, we'll assume the limit is not reached.
                    const count = redisClient ? await redisClient.incr(`notification:${users[i].userId}:${category}`) : 0;
                    if (dailyLimit && count > dailyLimit) currSilent = true;
                    // Send the notification if not silent
                    if (!currSilent) {
                        const payload = { body: currBody, icon: APP_ICON, link, title: currTitle };
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
            await prismaInstance.notification.createMany({
                data: users.map(({ userId }) => ({
                    category,
                    title: userTitles[userId],
                    description: userBodies[userId],
                    link,
                    imgLink: APP_ICON,
                    userId,
                })),
            });
        },
        trace: "0512",
    });
}

/**
 * Creates an appropriate label for a near-future event. 
 * Examples: (5 minutes, 2 hours, 1 day)
 * @param date The date of the event. If it's in the past, we return "now"
 */
function getEventStartLabel(date: Date) {
    const now = new Date();
    const diff = date.getTime() - now.getTime();
    if (diff < 0) return "now";
    if (diff < MINUTES_1_MS) return "in a few seconds";
    if (diff < HOURS_1_MS) return `in ${Math.round(diff / MINUTES_1_MS)} minutes`;
    if (diff < DAYS_1_MS) return `in ${Math.round(diff / HOURS_1_MS)} hours`;
    return `in ${Math.round(diff / DAYS_1_MS)} days`;
}

type Owner = { __typename: "User" | "Team", id: string };

type NotifyResultType = {
    toUser: (userId: string) => Promise<unknown>,
    toUsers: (userIds: (string | { userId: string, delays: number[] })[]) => Promise<unknown>,
    toTeam: (teamId: string, excludedUsers?: string[] | string) => Promise<unknown>,
    toOwner: (owner: { __typename: "User" | "Team", id: string }, excludedUsers?: string[] | string) => Promise<unknown>,
    toSubscribers: (objectType: SubscribableObject | `${SubscribableObject}`, objectId: string, excludedUsers?: string[] | string) => Promise<unknown>,
    toChatParticipants: (chatId: string, excludedUsers?: string[] | string) => Promise<unknown>,
    toAll: (objectType: GqlModelType | `${GqlModelType}`, objectId: string, owner: Owner | null | undefined, excludedUsers?: string | string[]) => Promise<unknown>,
}

/**
 * Replaces the label placeholders in a title and body's variables with the appropriate, translated labels.
 * @param bodyVariables The variables for the body
 * @param titleVariables The variables for the title
 * @param silent Whether the notification is silent
 * @param languages Preferred languages for error messages
 * @param users List of userIds and their languages
 * @returns PushToUser list with the variables replaced
 */
async function replaceLabels(
    bodyVariables: { [key: string]: string | number } | undefined,
    titleVariables: { [key: string]: string | number } | undefined,
    silent: boolean | undefined,
    languages: string[],
    users: Pick<PushToUser, "languages" | "userId" | "delays">[],
): Promise<PushToUser[]> {
    const labelRegex = /<Label\|([A-z]+):([0-9-]+)>/;
    // Initialize the result
    const result: PushToUser[] = users.map(u => ({ ...u, bodyVariables, titleVariables, silent }));
    // If titleVariables or bodyVariables contains "<Label|objectType:${id}>", we must inject 
    // the object's translated label
    let labelTranslations: { [key: string]: string } = {};
    // Helper function to query for label translations
    async function findTranslations(objectType: `${GqlModelType}`, objectId: string) {
        // Ignore if already translated
        if (Object.keys(labelTranslations).length > 0) return;
        const { dbTable, display } = ModelMap.getLogic(["dbTable", "display"], objectType, true, "replaceLabels 1");
        const labels = await (prismaInstance[dbTable] as PrismaDelegate).findUnique({
            where: { id: objectId },
            select: display().label.select(),
        });
        labelTranslations = labels ?? {};
    }
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
}

/**
 * Class returned by each notify function. Allows us to either
 * send the notification to one user, or to all admins of a team
 */
function NotifyResult(notification: NotifyResultParams): NotifyResultType {
    return {
        /**
         * Sends a notification to a user
         * @param userId The user's id
         */
        toUser: async (userId) => {
            const { bodyKey, bodyVariables, category, link, titleKey, titleVariables, silent, languages } = notification;
            // Shape and translate the notification for the user
            const users = await replaceLabels(bodyVariables, titleVariables, silent, languages, [{ languages, userId }]);
            // Send the notification
            await push({ bodyKey, category, link, titleKey, users });
        },
        /**
         * Sends a notification to multiple users
         * @param userIds The users' ids
         */
        toUsers: async (userIds) => {
            const { bodyKey, bodyVariables, category, link, titleKey, titleVariables, silent, languages } = notification;
            // Shape and translate the notification for each user
            const users = await replaceLabels(
                bodyVariables,
                titleVariables,
                silent,
                languages,
                userIds.map(data => ({
                    languages,
                    userId: typeof data === "string" ? data : data.userId,
                    delays: typeof data === "string" ? undefined : data.delays,
                })));
            // Send the notification to each user
            await push({ bodyKey, category, link, titleKey, users });
        },
        /**
         * Sends a notification to a team
         * @param teamId The team's id
         * @param excludedUsers IDs of users to exclude from the notification
         * (usually the user who triggered the notification)
         */
        toTeam: async (teamId, excludedUsers) => {
            const { bodyKey, bodyVariables, category, link, titleKey, titleVariables, silent, languages } = notification;
            // Find every admin of the team, excluding the user who triggered the notification
            const adminData = await ModelMap.get<TeamModelLogic>("Team").query.findAdminInfo(teamId, excludedUsers);
            // Shape and translate the notification for each admin
            const users = await replaceLabels(bodyVariables, titleVariables, silent, languages, adminData.map(({ id, languages }) => ({
                languages,
                userId: id,
            })));
            // Send the notification to each admin
            await push({ bodyKey, category, link, titleKey, users });
        },
        /**
         * Sends a notification to an owner of an object
         * @param owner The owner's id and __typename
         * @param excludedUsers IDs of users to exclude from the notification
         */
        toOwner: async (owner, excludedUsers) => {
            if (owner.__typename === "User" && !excludedUsers?.includes(owner.id)) {
                await NotifyResult(notification).toUser(owner.id);
            } else if (owner.__typename === "Team") {
                await NotifyResult(notification).toTeam(owner.id, excludedUsers);
            }
        },
        /**
         * Sends a notification to all subscribers of an object
         * @param objectType The __typename of object
         * @param objectId The object's id
         * @param excludedUsers IDs of users to exclude from the notification
         */
        toSubscribers: async (objectType, objectId, excludedUsers) => {
            try {
                const { bodyKey, bodyVariables, category, link, titleKey, titleVariables, silent, languages } = notification;
                await batch<Prisma.notification_subscriptionFindManyArgs>({
                    objectType: "NotificationSubscription",
                    processBatch: async (batch: { subscriberId: string, silent: boolean }[]) => {
                        // Shape and translate the notification for each subscriber
                        const users = await replaceLabels(bodyVariables, titleVariables, silent, languages, batch.map(({ subscriberId, silent }) => ({
                            languages,
                            silent,
                            userId: subscriberId,
                        })));
                        // Send the notification to each subscriber
                        await push({ bodyKey, category, link, titleKey, users });
                    },
                    select: { subscriberId: true, silent: true },
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
            } catch (error) {
                logger.error("Caught error in toSubscribers", { trace: "0467", error });
            }
        },
        /**
         * Sends a notification to all participants of a chat
         * @param chatId The chat's id
         * @param excludedUsers IDs of users to exclude from the notification
         */
        toChatParticipants: async (chatId, excludedUsers) => {
            try {
                const { bodyKey, bodyVariables, category, link, titleKey, titleVariables, silent, languages } = notification;
                await batch<Prisma.chat_participantsFindManyArgs>({
                    objectType: "ChatParticipant",
                    processBatch: async (batch: { user: { id: string } }[]) => {
                        // Shape and translate the notification for each participant
                        const users = await replaceLabels(bodyVariables, titleVariables, silent, languages, batch.map(({ user }) => ({
                            languages: ["en"], //TODO need to store user languages in db , then can update this
                            userId: user.id,
                        })));
                        // Send the notification to each participant
                        await push({ bodyKey, category, link, titleKey, users });
                    },
                    select: { user: { select: { id: true } } },
                    where: typeof excludedUsers === "string"
                        ? { AND: [{ chatId }, { userId: { not: excludedUsers } }] }
                        : Array.isArray(excludedUsers)
                            ? { AND: [{ chatId }, { userId: { notIn: excludedUsers } }] }
                            : { chatId },
                });
            } catch (error) {
                logger.error("Caught error in toChatParticipants", { trace: "0498", error });
            }
        },
        /**
         * Sends a notification to all relevant recipients:
         * - Owner (User or Team)
         * - Subscribers (if subscribable)
         * - Team members (if owner is a Team)
         * @param params Configuration for sending notifications
         */
        toAll: async (
            objectType,
            objectId,
            owner,
            excludedUsers,
        ) => {
            // If the object is subscribable, notify subscribers
            const isSubscribable = isObjectSubscribable(objectType);
            if (isSubscribable) {
                await NotifyResult(notification).toSubscribers(objectType as string as SubscribableObject, objectId, excludedUsers);
            }
            // If the object is not subscribable, for now we'll assume that it shouldn't trigger any notifications
            else {
                return;
            }
            // Notify the owner
            if (owner) {
                await NotifyResult(notification).toOwner(owner, excludedUsers);
            }
        },
    };
}

/**
 * Handles sending and registering notifications for a user or team. 
 * Team notifications are sent to every admin of the team.
 * Notifications settings and devices are queried from the main database.
 * Notification limits are tracked using Redis.
 */
export function Notify(languages: string[]) {
    return {
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
                const device = await prismaInstance.push_device.findUnique({
                    where: { endpoint },
                    select: { id: true },
                });
                // If it is, update the auth and p256dh keys
                if (device) {
                    logger.info(`device already registered: ${JSON.stringify(device)}`);
                    result = await prismaInstance.push_device.update({
                        where: { id: device.id },
                        data: { auth, p256dh, expires },
                        select,
                    });
                }
                // If it isn't, create a new device
                else {
                    result = await prismaInstance.push_device.create({
                        data: {
                            endpoint,
                            auth,
                            p256dh,
                            expires,
                            user: { connect: { id: userData.id } },
                        },
                        select,
                    });
                    logger.info(`device created: ${userData.id} ${JSON.stringify(result)}`);
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
            const device = await prismaInstance.push_device.findUnique({
                where: { id: deviceId },
                select: { userId: true },
            });
            if (!device || device.userId !== userId)
                throw new CustomError("0307", "PushDeviceNotFound", languages);
            // If it is, delete it  
            const deletedDevice = await prismaInstance.push_device.delete({ where: { id: deviceId } });
            return { __typename: "Success" as const, success: Boolean(deletedDevice) };
        },
        /**
         * Tests a push device by sending a test notification
         * @param deviceId The device's id
         * @param userId The user's id
         */
        testPushDevice: async ({
            auth,
            endpoint,
            p256dh,
        }: {
            auth: string
            endpoint: string,
            p256dh: string,
        }): Promise<Success> => {
            sendPush(
                { endpoint, keys: { p256dh, auth } },
                {
                    body: "This is a test notification",
                    icon: APP_ICON,
                    link: "https://vrooli.com/",
                    title: "Testing!",
                },
            );
            return { __typename: "Success" as const, success: true };
        },
        /**
         * Updates a user's notification settings
         * @param settings The new settings
         */
        updateSettings: async (settings: NotificationSettingsUpdateInput, userId: string) => {
            await updateNotificationSettings(settings, userId);
        },
        pushApiOutOfCredits: (): NotifyResultType => NotifyResult({
            bodyKey: "ApiOutOfCreditsBody",
            category: "AccountCreditsOrApi",
            languages,
            titleKey: "ApiOutOfCreditsTitle",
        }),
        pushAward: (awardName: string, awardDescription: string): NotifyResultType => NotifyResult({
            bodyKey: "AwardEarnedBody",
            bodyVariables: { awardName, awardDescription },
            category: "Award",
            languages,
            link: "/awards",
            titleKey: "AwardEarnedTitle",
        }),
        pushFreeCreditsReceived: (): NotifyResultType => NotifyResult({
            bodyKey: "FreeCreditsReceivedBody",
            category: "AccountCreditsOrApi",
            languages,
            titleKey: "FreeCreditsReceivedTitle",
        }),
        pushIssueStatusChange: (
            issueId: string,
            objectId: string,
            objectType: GqlModelType | `${GqlModelType}`,
            status: Exclude<IssueStatus, "Draft"> | `${Exclude<IssueStatus, "Draft">}`,
        ): NotifyResultType => NotifyResult({
            bodyKey: `IssueStatus${status}Body`,
            bodyVariables: { issueName: `<Label|Issue:${issueId}>`, objectName: `<Label|${objectType}:${objectId}>` },
            category: "IssueStatus",
            languages,
            link: `/issues/${issueId}`,
            titleKey: `IssueStatus${status}Title`,
        }),
        pushMessageReceived: (messageId: string, senderId: string): NotifyResultType => NotifyResult({
            bodyKey: "MessageReceivedBody",
            bodyVariables: { senderName: `<Label|User:${senderId}>` },
            category: "Message",
            languages,
            link: `/messages/${messageId}`,
            titleKey: "MessageReceivedTitle",
        }),
        pushNewDeviceSignIn: (): NotifyResultType => NotifyResult({
            bodyKey: "NewDeviceBody",
            category: "Security",
            languages,
            titleKey: "NewDeviceTitle",
        }),
        pushNewEmailVerification: (): NotifyResultType => NotifyResult({
            bodyKey: "NewEmailVerificationBody",
            category: "Security",
            languages,
            titleKey: "NewEmailVerificationTitle",
        }),
        pushNewQuestionOnObject: (objectType: `${GqlModelType}`, objectId: string, questionId: string): NotifyResultType => NotifyResult({
            bodyKey: "NewQuestionOnObjectBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "NewQuestionOrIssue",
            languages,
            link: `/questions/${questionId}`,
            titleKey: "NewQuestionOnObjectTitle",
        }),
        pushNewObjectInTeam: (objectType: `${GqlModelType}`, objectId: string, teamId: string): NotifyResultType => NotifyResult({
            bodyKey: "NewObjectInTeamBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, teamName: `<Label|Team:${teamId}>` },
            category: "NewObjectInTeam",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "NewObjectInTeamTitle",
            titleVariables: { teamName: `<Label|Team:${teamId}>` },
        }),
        pushNewObjectInProject: (objectType: `${GqlModelType}`, objectId: string, projectId: string): NotifyResultType => NotifyResult({
            bodyKey: "NewObjectInProjectBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, projectName: `<Label|Project:${projectId}>` },
            category: "NewObjectInProject",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "NewObjectInProjectTitle",
            titleVariables: { projectName: `<Label|Project:${projectId}>` },
        }),
        pushObjectReceivedBookmark: (objectType: `${GqlModelType}`, objectId: string, totalBookmarks: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedBookmarkBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalBookmarks },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "ObjectReceivedBookmarkTitle",
        }),
        pushObjectReceivedComment: (objectType: `${GqlModelType}`, objectId: string, totalComments: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedCommentBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalComments },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}?comments`,
            titleKey: "ObjectReceivedCommentTitle",
        }),
        pushObjectReceivedCopy: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "ObjectReceivedCopyTitle",
            titleVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        }),
        pushObjectReceivedUpvote: (objectType: `${GqlModelType}`, objectId: string, totalScore: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedUpvoteBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalScore },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
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
            titleKey: `ReportStatus${status}Title`,
        }),
        pushPullRequestStatusChange: (
            reportId: string,
            objectId: string,
            objectType: GqlModelType | `${GqlModelType}`,
            status: Exclude<PullRequestStatus, "Draft"> | `${Exclude<PullRequestStatus, "Draft">}`,
        ): NotifyResultType => NotifyResult({
            bodyKey: `PullRequestStatus${status}Body`,
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "PullRequestStatus",
            languages,
            link: `/${LINKS[objectType]}/${objectId}/pulls/${reportId}`,
            titleKey: `PullRequestStatus${status}Title`,
        }),
        pushQuestionActivity: (questionId: string): NotifyResultType => NotifyResult({
            bodyKey: "QuestionActivityBody",
            bodyVariables: { objectName: `<Label|Question:${questionId}>` },
            category: "QuestionActivity",
            languages,
            link: `/questions/${questionId}`,
            titleKey: "QuestionActivityTitle",
        }),
        pushRunStartedAutomatically: (runType: "RunProject" | "RunRoutine", runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunStartedAutomaticallyBody",
            bodyVariables: { runName: `<Label|${runType}:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunStartedAutomaticallyTitle",
        }),
        pushRunCompletedAutomatically: (runType: "RunProject" | "RunRoutine", runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunCompletedAutomaticallyBody",
            bodyVariables: { runName: `<Label|${runType}:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunCompletedAutomaticallyTitle",
        }),
        pushRunFailedAutomatically: (runType: "RunProject" | "RunRoutine", runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunFailedAutomaticallyBody",
            bodyVariables: { runName: `<Label|${runType}:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunFailedAutomaticallyTitle",
        }),
        pushScheduleReminder: (scheduleForId: string, scheduleForType: GqlModelType, startTime: Date): NotifyResultType => NotifyResult({
            bodyKey: "ScheduleUserBody",
            bodyVariables: { title: `<Label|${scheduleForType}:${scheduleForId}>`, startLabel: getEventStartLabel(startTime) },
            category: "Schedule",
            languages,
            link: scheduleForType === "Meeting" ? `/meeting/${scheduleForId}` : undefined,
        }),
        pushStreakReminder: (timeToReset: Date): NotifyResultType => NotifyResult({
            bodyKey: "StreakReminderBody",
            bodyVariables: { endLabel: getEventStartLabel(timeToReset) },
            category: "Streak",
            languages,
        }),
        pushStreakBroken: (): NotifyResultType => NotifyResult({
            bodyKey: "StreakBrokenBody",
            category: "Streak",
            languages,
            titleKey: "StreakBrokenTitle",
        }),
        pushTransferRequestSend: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRequestSendBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/transfers/${transferId}`,
            titleKey: "TransferRequestSendTitle",
        }),
        pushTransferRequestReceive: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRequestReceiveBody",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/transfers/${transferId}`,
            titleKey: "TransferRequestReceiveTitle",
        }),
        pushTransferAccepted: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferAcceptedTitle",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "TransferAcceptedTitle",
        }),
        pushTransferRejected: (objectType: `${GqlModelType}`, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRejectedTitle",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "TransferRejectedTitle",
        }),
        pushUserInvite: (friendUsername: string): NotifyResultType => NotifyResult({
            bodyKey: "UserInviteBody",
            bodyVariables: { friendUsername },
            category: "UserInvite",
            languages,
        }),
    };
}
