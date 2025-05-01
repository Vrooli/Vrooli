import { DAYS_1_MS, DEFAULT_LANGUAGE, DeferredDecisionData, generatePK, HOURS_1_MS, IssueStatus, LINKS, MINUTES_1_MS, ModelType, NotificationSettingsUpdateInput, PullRequestStatus, PushDevice, ReportStatus, SessionUser, SubscribableObject, Success } from "@local/shared";
import { Prisma } from "@prisma/client";
import i18next, { type TFuncKey } from "i18next";
import { InfoConverter } from "../builders/infoConverter.js";
import { type PartialApiInfo, type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { subscribableMapper } from "../events/subscriber.js";
import { ModelMap } from "../models/base/index.js";
import { PushDeviceModel } from "../models/base/pushDevice.js";
import { withRedis } from "../redisConn.js";
import { emitSocketEvent, roomHasOpenConnections } from "../sockets/events.js";
import { sendMail } from "../tasks/email/queue.js";
import { sendPush } from "../tasks/push/queue.js";
import { findRecipientsAndLimit, updateNotificationSettings } from "./notificationSettings.js";

/**
 * The icon of the app.
 */
const APP_ICON = "https://vrooli.com/apple-touch-icon.webp";

/**
 * When using the body for the title, this is the maximum number of characters to display.
 */
const MAX_BODY_TITLE_LENGTH = 10;

export type NotificationUrgency = "low" | "normal" | "critical";

export type NotificationCategory =
    "AccountCreditsOrApi" |
    "Award" |
    "IssueStatus" |
    "Message" |
    "NewObjectInTeam" |
    "NewObjectInProject" |
    "ObjectActivity" |
    "Promotion" |
    "PullRequestStatus" |
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
    languages: string[] | undefined,
    silent?: boolean,
    titleVariables?: { [key: string]: string | number }, // Variables can change depending on recipient's language
    userId: string,
}

type PushParams = {
    /** If provided, uses this body instead of the bodyKey */
    body?: string,
    /** Translation key for the body */
    bodyKey?: TransKey,
    /** Category of the notification */
    category: NotificationCategory,
    /** When the notification is clicked, this link will be opened */
    link?: string,
    /** If provided, uses this title instead of the titleKey */
    title?: string,
    /** Translation key for the title */
    titleKey?: TransKey,
    /** List of users to send the notification to */
    users: PushToUser[]
}

type NotifyResultParams = {
    /** If provided, uses this body instead of the bodyKey */
    body?: string,
    /** Translation key for the body */
    bodyKey?: TransKey,
    /** Variables for the body */
    bodyVariables?: { [key: string]: string | number },
    /** Category of the notification */
    category: NotificationCategory,
    /** Languages of the notification */
    languages: string[] | undefined,
    /** When the notification is clicked, this link will be opened */
    link?: string,
    /** If true, the notification will not be displayed (but the app will may be awakened in the background) */
    silent?: boolean,
    /** If provided, uses this title instead of the titleKey */
    title?: string,
    /** Translation key for the title */
    titleKey?: TransKey,
    /** Variables for the title */
    titleVariables?: { [key: string]: string | number },
}

/**
 * Checks if an object type can be subscribed to
 * @param objectType The object type to check
 * @returns True if the object type can be subscribed to
 */
export function isObjectSubscribable<T extends keyof typeof ModelType>(objectType: T): boolean {
    return objectType in SubscribableObject;
}

/**
 * Base function for pushing a notification to a list of users. If a user 
 * has no devices, emails, or phone numbers to send the message to, 
 * it is still stored in the database for the user to see when they 
 * open the app.
 */
async function push({
    body,
    bodyKey,
    category,
    link,
    title,
    titleKey,
    users,
}: PushParams) {
    // Find out which devices can receive this notification, and the daily limit
    const devicesAndLimits = await findRecipientsAndLimit(category, users.map(u => u.userId));
    // Find title and body for each user
    const userTitles: { [userId: string]: string } = {};
    const userBodies: { [userId: string]: string } = {};
    for (const user of users) {
        const lng = user.languages && user.languages.length > 0 ? user.languages[0] : DEFAULT_LANGUAGE;
        const bodyText: string | undefined =
            body
                ? body
                : bodyKey
                    ? i18next.t(`notify:${bodyKey}`, { lng, ...(user.bodyVariables ?? {}) })
                    : undefined;
        let titleText: string | undefined =
            title
                ? title
                : titleKey
                    ? i18next.t(`notify:${titleKey}`, { lng, ...(user.titleVariables ?? {}) })
                    : undefined;
        // If no title, use shortened body
        if (!titleText && bodyText) {
            titleText = `${bodyText.substring(0, MAX_BODY_TITLE_LENGTH)}...`;
        }
        // At least one of title or body must be defined
        if (!titleText && !bodyText) throw new CustomError("0362", "InternalError");
        userTitles[user.userId] = titleText ?? "";
        userBodies[user.userId] = bodyText ?? "";
    }
    await withRedis({
        process: async (redisClient) => {
            // For each user
            for (let i = 0; i < users.length; i++) {
                // For each delay
                for (const delay of users[i].delays ?? [0]) {
                    const { pushDevices, emails, dailyLimit } = devicesAndLimits[i];
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
            // Store the notifications in the database and emit via WebSocket
            const notificationsData = users.map(({ userId }) => ({
                category,
                description: userBodies[userId],
                id: generatePK(),
                imgLink: APP_ICON,
                link,
                title: userTitles[userId],
                userId: BigInt(userId),
            }));

            // Create all notifications
            await DbProvider.get().notification.createMany({
                data: notificationsData,
            });

            // Emit via WebSocket for connected users
            for (const user of users) {
                if (roomHasOpenConnections(user.userId)) {
                    // Find the notification we just created for this user
                    const notification = notificationsData.find(n => n.userId.toString() === user.userId);
                    if (notification) {
                        emitSocketEvent("notification", user.userId, {
                            ...notification,
                            id: notification.id.toString(),
                            __typename: "Notification",
                            createdAt: new Date().toISOString(),
                            isRead: false,
                        });
                    }
                }
            }
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
    toAll: (objectType: ModelType | `${ModelType}`, objectId: string, owner: Owner | null | undefined, excludedUsers?: string | string[]) => Promise<unknown>,
}

/**
 * Replaces the label placeholders in a title and body's variables with the appropriate, translated labels.
 * @param bodyVariables The variables for the body
 * @param titleVariables The variables for the title
 * @param silent Whether the notification is silent
 * @param users List of userIds and their languages
 * @returns PushToUser list with the variables replaced
 */
async function replaceLabels(
    bodyVariables: { [key: string]: string | number } | undefined,
    titleVariables: { [key: string]: string | number } | undefined,
    silent: boolean | undefined,
    users: Pick<PushToUser, "languages" | "userId" | "delays">[],
): Promise<PushToUser[]> {
    const labelRegex = /<Label\|([A-z]+):([0-9-]+)>/;
    // Initialize the result
    const result: PushToUser[] = users.map(u => ({ ...u, bodyVariables, titleVariables, silent }));
    // If titleVariables or bodyVariables contains "<Label|objectType:${id}>", we must inject 
    // the object's translated label
    let labelTranslations: { [key: string]: string } = {};
    // Helper function to query for label translations
    async function findTranslations(objectType: `${ModelType}`, objectId: string) {
        // Ignore if already translated
        if (Object.keys(labelTranslations).length > 0) return;
        const { dbTable, display } = ModelMap.getLogic(["dbTable", "display"], objectType, true, "replaceLabels 1");
        const labels = await (DbProvider.get()[dbTable] as PrismaDelegate).findUnique({
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
                    await findTranslations(match[1] as ModelType, match[2]);
                    // In each params, replace the matching substring with the label
                    const { display } = ModelMap.getLogic(["display"], match[1] as ModelType, true, "replaceLabels 2");
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
                    await findTranslations(match[1] as ModelType, match[2]);
                    // In each params, replace the matching substring with the label
                    const { display } = ModelMap.getLogic(["display"], match[1] as ModelType, true, "replaceLabels 3");
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
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent, languages } = notification;
            // Shape and translate the notification for the user
            const users = await replaceLabels(bodyVariables, titleVariables, silent, [{ languages, userId }]);
            // Send the notification
            await push({ body, bodyKey, category, link, title, titleKey, users });
        },
        /**
         * Sends a notification to multiple users
         * @param userIds The users' ids
         */
        toUsers: async (userIds) => {
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent, languages } = notification;
            // Shape and translate the notification for each user
            const users = await replaceLabels(
                bodyVariables,
                titleVariables,
                silent,
                userIds.map(data => ({
                    languages,
                    userId: typeof data === "string" ? data : data.userId,
                    delays: typeof data === "string" ? undefined : data.delays,
                })));
            // Send the notification to each user
            await push({ body, bodyKey, category, link, title, titleKey, users });
        },
        /**
         * Sends a notification to a team
         * @param teamId The team's id
         * @param excludedUsers IDs of users to exclude from the notification
         * (usually the user who triggered the notification)
         */
        toTeam: async (teamId, excludedUsers) => {
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent } = notification;
            // Find every admin of the team, excluding the user who triggered the notification
            const adminData = await DbProvider.get().member.findMany({
                where: {
                    teamId: BigInt(teamId),
                    isAdmin: true,
                    ...(typeof excludedUsers === "string" ? [{ userId: { not: BigInt(excludedUsers) } }] : Array.isArray(excludedUsers) ? [{ userId: { notIn: excludedUsers.map(id => BigInt(id)) } }] : []),
                },
                select: {
                    user: {
                        select: {
                            id: true,
                            languages: true,
                        },
                    },
                },
            });
            const admins = adminData.map(({ user }) => user);
            // Shape and translate the notification for each admin
            const users = await replaceLabels(bodyVariables, titleVariables, silent, admins.map(({ id, languages }) => ({
                languages,
                userId: id.toString(),
            })));
            // Send the notification to each admin
            await push({ body, bodyKey, category, link, title, titleKey, users });
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
                const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent, languages } = notification;
                const { batch } = await import("../utils/batch.js");
                await batch<Prisma.notification_subscriptionFindManyArgs>({
                    objectType: "NotificationSubscription",
                    processBatch: async (batch: { subscriberId: string, silent: boolean }[]) => {
                        // Shape and translate the notification for each subscriber
                        const users = await replaceLabels(bodyVariables, titleVariables, silent, batch.map(({ subscriberId, silent }) => ({
                            languages,
                            silent,
                            userId: subscriberId,
                        })));
                        // Send the notification to each subscriber
                        await push({ body, bodyKey, category, link, title, titleKey, users });
                    },
                    select: { subscriberId: true, silent: true },
                    where: {
                        AND: [
                            { [subscribableMapper[objectType]]: { id: objectId } },
                            ...(typeof excludedUsers === "string" ?
                                [{ subscriberId: { not: BigInt(excludedUsers) } }] :
                                Array.isArray(excludedUsers) ?
                                    [{ subscriberId: { notIn: excludedUsers.map(id => BigInt(id)) } }] :
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
                const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent } = notification;
                const { batch } = await import("../utils/batch.js");
                await batch<Prisma.chat_participantsFindManyArgs>({
                    objectType: "ChatParticipant",
                    processBatch: async (batch: { user: { id: string, languages: string[] } }[]) => {
                        // Shape and translate the notification for each participant
                        const users = await replaceLabels(bodyVariables, titleVariables, silent, batch.map(({ user }) => ({
                            languages: user.languages,
                            userId: user.id,
                        })));
                        // Send the notification to each participant
                        await push({ body, bodyKey, category, link, title, titleKey, users });
                    },
                    select: { user: { select: { id: true, languages: true } } },
                    where: typeof excludedUsers === "string"
                        ? { AND: [{ chatId: BigInt(chatId) }, { userId: { not: BigInt(excludedUsers) } }] }
                        : Array.isArray(excludedUsers)
                            ? { AND: [{ chatId: BigInt(chatId) }, { userId: { notIn: excludedUsers.map(id => BigInt(id)) } }] }
                            : { chatId: BigInt(chatId) },
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
export function Notify(languages: string[] | undefined) {
    return {
        /** Sets up a push device to receive notifications */
        registerPushDevice: async ({ endpoint, p256dh, auth, expires, userData, info }: {
            endpoint: string,
            p256dh: string,
            auth: string,
            expires?: Date,
            name?: string,
            userData: SessionUser,
            info: PartialApiInfo,
        }): Promise<PushDevice> => {
            const partialInfo = InfoConverter.get().fromApiToPartialApi(info, PushDeviceModel.format.apiRelMap, true);
            let select: { [key: string]: any } | undefined;
            let result: any = {};
            try {
                select = InfoConverter.get().fromPartialApiToPrismaSelect(partialInfo)?.select;
                // Check if the device is already registered
                const device = await DbProvider.get().push_device.findUnique({
                    where: { endpoint },
                    select: { id: true },
                });
                // If it is, update the auth and p256dh keys
                if (device) {
                    logger.info(`device already registered: ${JSON.stringify(device)}`);
                    result = await DbProvider.get().push_device.update({
                        where: { id: device.id },
                        data: { auth, p256dh, expires },
                        select,
                    });
                }
                // If it isn't, create a new device
                else {
                    result = await DbProvider.get().push_device.create({
                        data: {
                            endpoint,
                            auth,
                            p256dh,
                            expires,
                            user: { connect: { id: BigInt(userData.id) } },
                        },
                        select,
                    });
                    logger.info(`device created: ${userData.id} ${JSON.stringify(result)}`);
                }
                return result;
            } catch (error) {
                throw new CustomError("0452", "InternalError", { error, select, result });
            }
        },
        /**
         * Removes a push device from the database
         * @param deviceId The device's id
         * @param userId The user's id
         */
        unregisterPushDevice: async (deviceId: string, userId: string): Promise<Success> => {
            // Check if the device is registered to the user
            const device = await DbProvider.get().push_device.findUnique({
                where: { id: BigInt(deviceId) },
                select: { userId: true },
            });
            if (!device || device.userId.toString() !== userId)
                throw new CustomError("0307", "PushDeviceNotFound");
            // If it is, delete it  
            const deletedDevice = await DbProvider.get().push_device.delete({ where: { id: BigInt(deviceId) } });
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
            bodyKey: "ApiOutOfCredits_Body",
            category: "AccountCreditsOrApi",
            languages,
            titleKey: "ApiOutOfCredits_Title",
        }),
        pushAward: (awardName: string, awardDescription: string): NotifyResultType => NotifyResult({
            bodyKey: "AwardEarned_Body",
            bodyVariables: { awardName, awardDescription },
            category: "Award",
            languages,
            link: "/awards",
            titleKey: "AwardEarned_Title",
        }),
        pushFreeCreditsReceived: (): NotifyResultType => NotifyResult({
            bodyKey: "FreeCreditsReceived_Body",
            category: "AccountCreditsOrApi",
            languages,
            titleKey: "FreeCreditsReceived_Title",
        }),
        pushIssueStatusChange: (
            issueId: string,
            objectId: string,
            objectType: ModelType | `${ModelType}`,
            status: Exclude<IssueStatus, "Draft"> | `${Exclude<IssueStatus, "Draft">}`,
        ): NotifyResultType => NotifyResult({
            bodyKey: `IssueStatus${status}_Body`,
            bodyVariables: { issueName: `<Label|Issue:${issueId}>`, objectName: `<Label|${objectType}:${objectId}>` },
            category: "IssueStatus",
            languages,
            link: `/issues/${issueId}`,
            titleKey: `IssueStatus${status}_Title`,
        }),
        pushMessageReceived: (messageId: string, senderId: string): NotifyResultType => NotifyResult({
            bodyKey: "MessageReceived_Body",
            bodyVariables: { senderName: `<Label|User:${senderId}>` },
            category: "Message",
            languages,
            link: `/messages/${messageId}`,
            titleKey: "MessageReceived_Title",
        }),
        pushNewDecisionRequest: (decision: DeferredDecisionData, runId: string): NotifyResultType => NotifyResult({
            body: decision.message,
            bodyKey: "NewDecisionRequest_Body",
            bodyVariables: { objectName: `<Label|Run:${runId}>` },
            category: "Run",
            languages,
            titleKey: "NewDecisionRequest_Title",
        }),
        pushNewDeviceSignIn: (): NotifyResultType => NotifyResult({
            bodyKey: "NewDevice_Body",
            category: "Security",
            languages,
            titleKey: "NewDevice_Title",
        }),
        pushNewEmailVerification: (): NotifyResultType => NotifyResult({
            bodyKey: "NewEmailVerification_Body",
            category: "Security",
            languages,
            titleKey: "NewEmailVerification_Title",
        }),
        pushNewObjectInTeam: (objectType: `${ModelType}`, objectId: string, teamId: string): NotifyResultType => NotifyResult({
            bodyKey: "NewObjectInTeam_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, teamName: `<Label|Team:${teamId}>` },
            category: "NewObjectInTeam",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "NewObjectInTeam_Title",
            titleVariables: { teamName: `<Label|Team:${teamId}>` },
        }),
        pushNewObjectInProject: (objectType: `${ModelType}`, objectId: string, projectId: string): NotifyResultType => NotifyResult({
            bodyKey: "NewObjectInProject_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, projectName: `<Label|Project:${projectId}>` },
            category: "NewObjectInProject",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "NewObjectInProject_Title",
            titleVariables: { projectName: `<Label|Project:${projectId}>` },
        }),
        pushObjectReceivedBookmark: (objectType: `${ModelType}`, objectId: string, totalBookmarks: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedBookmark_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalBookmarks },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "ObjectReceivedBookmark_Title",
        }),
        pushObjectReceivedComment: (objectType: `${ModelType}`, objectId: string, totalComments: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedComment_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalComments },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}?comments`,
            titleKey: "ObjectReceivedComment_Title",
        }),
        pushObjectReceivedCopy: (objectType: `${ModelType}`, objectId: string): NotifyResultType => NotifyResult({
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "ObjectReceivedCopy_Title",
            titleVariables: { objectName: `<Label|${objectType}:${objectId}>` },
        }),
        pushObjectReceivedUpvote: (objectType: `${ModelType}`, objectId: string, totalScore: number): NotifyResultType => NotifyResult({
            bodyKey: "ObjectReceivedUpvote_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>`, count: totalScore },
            category: "ObjectActivity",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "ObjectReceivedUpvote_Title",
        }),
        /**
         * NOTE: If object is being deleted, this should be called before the object is deleted. 
         * Otherwise, we cannot display the name of the object in the notification.
         */
        pushReportStatusChange: (
            reportId: string,
            objectId: string,
            objectType: ModelType | `${ModelType}`,
            status: ReportStatus | `${ReportStatus}`,
        ): NotifyResultType => NotifyResult({
            bodyKey: `ReportStatus${status}_Body`,
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "ReportStatus",
            languages,
            link: `/${LINKS[objectType]}/${objectId}/reports/${reportId}`,
            titleKey: `ReportStatus${status}_Title`,
        }),
        pushPullRequestStatusChange: (
            reportId: string,
            objectId: string,
            objectType: ModelType | `${ModelType}`,
            status: Exclude<PullRequestStatus, "Draft"> | `${Exclude<PullRequestStatus, "Draft">}`,
        ): NotifyResultType => NotifyResult({
            bodyKey: `PullRequestStatus${status}_Body`,
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "PullRequestStatus",
            languages,
            link: `/${LINKS[objectType]}/${objectId}/pulls/${reportId}`,
            titleKey: `PullRequestStatus${status}_Title`,
        }),
        pushRunStartedAutomatically: (runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunStartedAutomatically_Body",
            bodyVariables: { runName: `<Label|Run:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunStartedAutomatically_Title",
        }),
        pushRunCompletedAutomatically: (runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunCompletedAutomatically_Body",
            bodyVariables: { runName: `<Label|Run:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunCompletedAutomatically_Title",
        }),
        pushRunFailedAutomatically: (runId: string): NotifyResultType => NotifyResult({
            bodyKey: "RunFailedAutomatically_Body",
            bodyVariables: { runName: `<Label|Run:${runId}>` },
            category: "Run",
            languages,
            link: `/runs/${runId}`,
            titleKey: "RunFailedAutomatically_Title",
        }),
        pushScheduleReminder: (scheduleForId: string, scheduleForType: ModelType, startTime: Date): NotifyResultType => NotifyResult({
            bodyKey: "ScheduleUser_Body",
            bodyVariables: { title: `<Label|${scheduleForType}:${scheduleForId}>`, startLabel: getEventStartLabel(startTime) },
            category: "Schedule",
            languages,
            link: scheduleForType === "Meeting" ? `/meeting/${scheduleForId}` : undefined,
        }),
        pushStreakReminder: (timeToReset: Date): NotifyResultType => NotifyResult({
            bodyKey: "StreakReminder_Body",
            bodyVariables: { endLabel: getEventStartLabel(timeToReset) },
            category: "Streak",
            languages,
        }),
        pushStreakBroken: (): NotifyResultType => NotifyResult({
            bodyKey: "StreakBroken_Body",
            category: "Streak",
            languages,
            titleKey: "StreakBroken_Title",
        }),
        pushTransferRequestSend: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRequestSend_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/transfers/${transferId}`,
            titleKey: "TransferRequestSend_Title",
        }),
        pushTransferRequestReceive: (transferId: string, objectType: string, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRequestReceive_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/transfers/${transferId}`,
            titleKey: "TransferRequestReceive_Title",
        }),
        pushTransferAccepted: (objectType: `${ModelType}`, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferAccepted_Title",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "TransferAccepted_Title",
        }),
        pushTransferRejected: (objectType: `${ModelType}`, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRejected_Title",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "TransferRejected_Title",
        }),
        pushUserInvite: (friendUsername: string): NotifyResultType => NotifyResult({
            bodyKey: "UserInvite_Body",
            bodyVariables: { friendUsername },
            category: "UserInvite",
            languages,
        }),
    };
}
