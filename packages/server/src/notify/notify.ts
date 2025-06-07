import { DAYS_1_MS, DAYS_1_S, DEFAULT_LANGUAGE, type DeferredDecisionData, endpointsTask, generatePK, HOURS_1_MS, type IssueStatus, LINKS, MINUTES_1_MS, type ModelType, nanoid, type NotificationSettingsUpdateInput, type PullRequestStatus, type PushDevice, type ReportStatus, SECONDS_1_MS, type SessionUser, SubscribableObject, type Success } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import i18next, { type TFuncKey } from "i18next";
import { type Cluster, type Redis } from "ioredis";
import { InfoConverter } from "../builders/infoConverter.js";
import { type PartialApiInfo, type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { subscribableMapper } from "../events/subscriber.js";
import { ModelMap } from "../models/base/index.js";
import { PushDeviceModel } from "../models/base/pushDevice.js";
import { CacheService } from "../redisConn.js";
import { SocketService } from "../sockets/io.js";
import type { BaseTaskData, ManagedQueue } from "../tasks/queueFactory.js";
import { QueueService } from "../tasks/queues.js";
import { QueueTaskType } from "../tasks/taskTypes.js";
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

/** Defines how a notification should be delivered, balancing push and WebSocket. */
export type NotificationDeliveryMode =
    /** 
     * Default behavior: Sends push if user is offline. 
     * WebSocket event (via notificationCreateProcess) occurs if user is online. 
     * Notification is always persisted. 
     */
    "default" |
    /** 
     * Forces a push notification regardless of connection status. 
     * WebSocket event also occurs if user is online. 
     * Notification is always persisted. 
     */
    "force_push" |
    /** 
     * Prefers WebSocket if user is connected (no push). 
     * Sends push if user is offline. 
     * Notification is always persisted. 
     */
    "prefer_websocket";

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
    "SystemAlert" |
    "Transfer" |
    "UserInvite";

type TransKey = TFuncKey<"notify", undefined>

/**
 * Configuration for sending a notification to a specific user.
 * This type is used to customize notification content and delivery for individual recipients.
 */
type PushToUser = {
    /**
     * Variables to be interpolated into the notification body text.
     * These can change depending on the recipient's language.
     * Example: { userName: "John", count: 5 }
     */
    bodyVariables?: { [key: string]: string | number },
    /**
     * List of languages preferred by the user, in order of preference.
     * Used to determine which translation to use for the notification.
     * If undefined, the system default language will be used.
     */
    languages: string[] | undefined,
    /**
     * If true, the notification will not be displayed to the user.
     * The notification will still be stored in the database but won't trigger
     * any visual or audio alerts.
     */
    silent?: boolean,
    /**
     * Variables to be interpolated into the notification title text.
     * These can change depending on the recipient's language.
     * Example: { userName: "John", count: 5 }
     */
    titleVariables?: { [key: string]: string | number },
    /**
     * The unique identifier of the user who should receive the notification.
     */
    userId: string,
    /**
     * Array of delay times in milliseconds for scheduling multiple notifications.
     * Each delay value will create a separate notification task.
     * 
     * Common use cases:
     * - Event reminders: [300000, 3600000] (5 minutes and 1 hour before)
     * - Scheduled notifications: [86400000] (24 hours from now)
     * 
     * Notes:
     * - Only positive numbers are considered valid
     * - Invalid values (non-numbers, negative numbers) are ignored
     * - Values are sorted in ascending order before processing
     * - If empty or undefined, notification is sent immediately
     */
    delays?: number[],
    /** Optional delivery mode preference. Defaults to "default". */
    deliveryMode?: NotificationDeliveryMode;
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
    /** Raw timestamp for schedule start label */
    rawStartTs?: number,
    /** Raw timestamp for streak end label */
    rawEndTs?: number,
}

const notificationSubscriptionSelect = {
    subscriberId: true,
    silent: true,
    subscriber: { select: { languages: true } },
} as const;

type NotificationSubscriptionPayload = Prisma.notification_subscriptionGetPayload<{ select: typeof notificationSubscriptionSelect }>;

// FIRST_EDIT: declare select shape and payload type for chat participants
const chatParticipantsSelect = {
    user: { select: { id: true, languages: true } },
} as const;

type ChatParticipantPayload = Prisma.chat_participantsGetPayload<{ select: typeof chatParticipantsSelect }>;

/**
 * Checks if an object type can be subscribed to
 * @param objectType The object type to check
 * @returns True if the object type can be subscribed to
 */
export function isObjectSubscribable<T extends keyof typeof ModelType>(objectType: T): boolean {
    return objectType in SubscribableObject;
}

/**
 * Enqueues a task one or more times based on delays.
 * @param taskQueue The queue to enqueue the task to
 * @param task The task to enqueue
 * @param delays The delays to enqueue the task at. If none are provided, the task is enqueued immediately.
 * @returns The IDs of the tasks that were enqueued.
 */
async function enqueueTaskWithDelays<Data extends BaseTaskData>(
    taskQueue: ManagedQueue<Data>,
    task: Omit<Data, "id">,
    delays: number[],
): Promise<string[]> {
    const taskIds: string[] = [];
    if (delays.length === 0) {
        const taskId = generatePK().toString();
        taskIds.push(taskId);
        await taskQueue.add({
            ...task,
            id: taskId,
        } as Data);
    }
    for (const delay of delays) {
        const taskId = generatePK().toString();
        taskIds.push(taskId);
        await taskQueue.add({
            ...task,
            id: taskId,
        } as Data, { delay });
    }
    return taskIds;
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

    let redisClient: Redis | Cluster | undefined;
    try {
        redisClient = await CacheService.get().raw();
    } catch (error) {
        logger.error("[Notify] Failed to get Redis client from CacheService for incrementing notification counts. Proceeding without limits.", { trace: "notify-push-redis-fail", error: error instanceof Error ? error.message : String(error) });
        // redisClient remains undefined, and the loop below will use count = 0
    }

    // For each user
    for (let i = 0; i < users.length; i++) {
        const currentUser = users[i]; // current user from the loop
        const { pushDevices, emails, dailyLimit } = devicesAndLimits[i]; // devicesAndLimits corresponds to users[i]
        let effectiveSilent = currentUser.silent ?? false; // Start with user's preference, or default to false
        const effectiveDeliveryMode = currentUser.deliveryMode || "default"; // Default to "default" if not specified
        const currTitle = userTitles[currentUser.userId];
        const currBody = userBodies[currentUser.userId];

        // Check and apply daily notification limits if a limit is defined for the category
        // and the notification isn't already marked silent by the user.
        if (dailyLimit && !effectiveSilent) {
            let currentRedisCount = 0; // Will store the count from Redis
            let redisLimitCheckSuccessful = false;

            if (redisClient) {
                try {
                    const key = `notification:${currentUser.userId}:${category}`;
                    // Increment and get the new count. Assign to currentRedisCount for use outside this block.
                    currentRedisCount = await redisClient.incr(key);
                    // Only set expiration if this is the first increment for the period
                    if (currentRedisCount === 1) {
                        await redisClient.expire(key, DAYS_1_S);
                    }
                    redisLimitCheckSuccessful = true; // Mark as successful only if all Redis operations complete
                } catch (error) {
                    logger.error(`[Notify] Failed to increment/expire notification count in Redis for user ${currentUser.userId}, category ${category}. Assuming limit might be exceeded.`, { trace: "notify-push-incr-expire-fail", userId: currentUser.userId, category, error: error instanceof Error ? error.message : String(error) });
                    // redisLimitCheckSuccessful remains false, will lead to silencing if dailyLimit is active
                }
            } else {
                // redisClient is undefined (e.g., CacheService.get().raw() failed)
                logger.warn(`[Notify] Redis client unavailable for notification count for user ${currentUser.userId}, category ${category}. Assuming limit might be exceeded.`, { trace: "notify-push-redis-unavailable", userId: currentUser.userId, category });
                // redisLimitCheckSuccessful remains false, will lead to silencing if dailyLimit is active
            }

            if (redisLimitCheckSuccessful) {
                // Check if the current count (after incrementing) exceeds the daily limit
                if (currentRedisCount > dailyLimit) {
                    effectiveSilent = true; // Limit exceeded, silence the notification
                    logger.info(`[Notify] Daily notification limit (${dailyLimit}) exceeded for user ${currentUser.userId}, category ${category}. Current count: ${currentRedisCount}. Notification silenced.`, { trace: "notify-push-limit-exceeded", userId: currentUser.userId, category, currentRedisCount, dailyLimit });
                }
            } else {
                // INTENTIONAL: If Redis operations fail (either client unavailable or incr/expire failed)
                // and a dailyLimit is active for the category, the notification is made silent.
                // This is a deliberate choice to prevent users, who might otherwise have notifications
                // silenced by their daily limit, from being spammed if the limiting mechanism itself is down.
                // It prioritizes preventing potential spam over ensuring delivery when the limiter fails.
                logger.warn(`[Notify] Notification for user ${currentUser.userId}, category ${category} made silent due to Redis issues and active dailyLimit.`, { trace: "notify-push-limit-enforced-silent-on-redis-failure", userId: currentUser.userId, category });
                effectiveSilent = true;
            }
        }

        // Process delays if specified
        const validDelays: number[] = [];
        if (currentUser.delays && currentUser.delays.length > 0) {
            // Filter and validate delays
            validDelays.push(...currentUser.delays
                .filter((delay): delay is number => typeof delay === "number" && delay > 0)
                .sort((a, b) => a - b)); // Sort delays in ascending order

            if (validDelays.length === 0) {
                logger.warn(`[Notify] No valid positive delay values found for user ${currentUser.userId}, category ${category}. Notification will be sent without delay if not otherwise silent.`, {
                    trace: "notify-push-no-valid-delays",
                    userId: currentUser.userId,
                    category,
                    specifiedDelays: currentUser.delays,
                });
            } else {
                logger.info(`[Notify] Scheduling ${validDelays.length} push notification(s) for user ${currentUser.userId} (category: ${category}) with delays: ${validDelays.join(", ")}ms`, {
                    trace: "notify-push-delays-scheduled",
                    userId: currentUser.userId,
                    category,
                    delays: validDelays,
                });
            }
        }

        // Always schedule the in-app notification record creation for DB persistence.
        try {
            await enqueueTaskWithDelays(QueueService.get().notification, {
                type: QueueTaskType.NOTIFICATION_CREATE,
                userId: currentUser.userId,
                category,
                title: currTitle,
                description: currBody,
                link,
                imgLink: APP_ICON,
                sendWebSocketEvent: !effectiveSilent, // Send WebSocket event if not silent
            }, validDelays);
        } catch (err) {
            logger.error("Error scheduling in-app notification record creation for DB persistence", {
                trace: "notify-push-db-record-scheduling-fail",
                error: err instanceof Error ? err.message : String(err),
                userId: currentUser.userId,
                category,
            });
        }

        // Conditionally send actual push notifications based on deliveryMode and silence status
        if (!effectiveSilent) {
            const userIsConnected = SocketService.get().roomHasOpenConnections(currentUser.userId);
            let sendPushNotification = false;

            if (effectiveDeliveryMode === "force_push") {
                sendPushNotification = true;
            } else if ((effectiveDeliveryMode === "default" || effectiveDeliveryMode === "prefer_websocket") && !userIsConnected) {
                sendPushNotification = true;
            } else if (effectiveDeliveryMode === "prefer_websocket" && userIsConnected) {
                // User is connected and mode is prefer_websocket, so no push needed.
                // The WebSocket event will be handled by notificationCreateProcess.
                sendPushNotification = false;
                logger.info(`[Notify] User ${currentUser.userId} is connected, deliveryMode is 'prefer_websocket'. Skipping push notification, relying on WebSocket.`, {
                    userId: currentUser.userId, category,
                });
            }

            if (sendPushNotification) {
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
                        await enqueueTaskWithDelays(QueueService.get().push, {
                            type: QueueTaskType.PUSH_NOTIFICATION,
                            subscription,
                            payload,
                        }, validDelays);
                    } catch (err) {
                        logger.error("Error sending push notification", {
                            trace: "0306",
                            error: err instanceof Error ? err.message : String(err),
                            userId: currentUser.userId,
                            category,
                            delays: validDelays,
                        });
                    }
                }
                // Send to each email (ignore if no title)
                if (emails.length && currTitle) {
                    try {
                        await QueueService.get().email.addTask({
                            type: QueueTaskType.EMAIL_SEND,
                            id: nanoid(),
                            to: emails.map(e => e.emailAddress),
                            subject: currTitle,
                            text: currBody,
                            html: "", // TODO: Consider HTML email template
                        });
                    } catch (err) {
                        logger.error("Error sending email notification", {
                            trace: "notify-push-email-send-fail",
                            error: err instanceof Error ? err.message : String(err),
                            userId: currentUser.userId,
                            category,
                        });
                    }
                }
            }
        }
    }
}

/**
 * Creates a localized label for a near-future event.
 * @param date The date of the event. If it's in the past, returns "now" in the target language.
 * @param lng  The target language code (e.g. "en", "es").
 */
function getEventStartLabel(date: Date, lng: string) {
    const now = new Date();
    const diff = date.getTime() - now.getTime();
    if (diff < 0) {
        return i18next.t("notify:Time_Now", { lng });
    }
    if (diff < MINUTES_1_MS) {
        return i18next.t("notify:Time_FewSeconds", { lng });
    }
    const minutes = Math.ceil(diff / MINUTES_1_MS);
    if (diff < HOURS_1_MS) {
        return i18next.t("notify:Time_Minutes", { lng, count: minutes });
    }
    const hours = Math.ceil(diff / HOURS_1_MS);
    if (diff < DAYS_1_MS) {
        return i18next.t("notify:Time_Hours", { lng, count: hours });
    }
    const days = Math.ceil(diff / DAYS_1_MS);
    return i18next.t("notify:Time_Days", { lng, count: days });
}

type Owner = { __typename: "User" | "Team", id: string };

type NotifyResultType = {
    toUser: (userId: string, deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toUsers: (userIds: (string | { userId: string, delays: number[] })[], deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toTeam: (teamId: string, excludedUsers?: string[] | string, deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toOwner: (owner: { __typename: "User" | "Team", id: string }, excludedUsers?: string[] | string, deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toSubscribers: (objectType: SubscribableObject | `${SubscribableObject}`, objectId: string, excludedUsers?: string[] | string, deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toChatParticipants: (chatId: string, excludedUsers?: string[] | string, deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
    toAll: (objectType: ModelType | `${ModelType}`, objectId: string, owner: Owner | null | undefined, excludedUsers?: string | string[], deliveryMode?: NotificationDeliveryMode) => Promise<unknown>,
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
    users: PushToUser[],
): Promise<PushToUser[]> {
    const labelRegex = /<Label\|([A-Za-z]+):([0-9]+)>/g; // More specific regex
    const uniqueLabelsToFetch = new Map<string, { objectType: ModelType, objectId: string }>();

    // --- Step 1: Gather all unique labels from all users and all variables ---
    const result: PushToUser[] = users.map(u => ({
        ...u,
        // Deep copy variables to ensure each user's variables can be independently modified.
        bodyVariables: bodyVariables ? { ...bodyVariables } : undefined,
        titleVariables: titleVariables ? { ...titleVariables } : undefined,
        // Preserve per-user silent preference if provided, otherwise use notification-level silent.
        silent: u.silent ?? silent,
    }));

    for (const currentUserResult of result) {
        // eslint-disable-next-line func-style
        const processVariablesForDiscovery = (vars: { [key: string]: string | number } | undefined) => {
            if (!vars) return;
            for (const key of Object.keys(vars)) {
                const value = vars[key];
                if (typeof value === "string") {
                    let match;
                    labelRegex.lastIndex = 0; // Reset regex state for global regex before each new string
                    while ((match = labelRegex.exec(value)) !== null) {
                        const objectType = match[1] as ModelType;
                        const objectId = match[2];
                        const labelKey = `${objectType}:${objectId}`;
                        if (!uniqueLabelsToFetch.has(labelKey)) {
                            uniqueLabelsToFetch.set(labelKey, { objectType, objectId });
                        }
                    }
                }
            }
        };
        processVariablesForDiscovery(currentUserResult.titleVariables);
        processVariablesForDiscovery(currentUserResult.bodyVariables);
    }

    // --- Step 2: Batch fetch all translations for unique labels ---
    const fetchedLabelData = new Map<string, any>(); // Stores raw label data from DB (the result of display().label.select())

    // Pre-fetch all unique labels
    // Using Promise.all to fetch in parallel for potentially better performance
    await Promise.all(
        Array.from(uniqueLabelsToFetch.values()).map(async ({ objectType, objectId }) => {
            const labelKey = `${objectType}:${objectId}`;
            try {
                const { dbTable, display } = ModelMap.getLogic(["dbTable", "display"], objectType, true, `fetchSpecificLabelTranslations for ${objectType}:${objectId}`);
                const labelDbResult = await (DbProvider.get()[dbTable] as PrismaDelegate).findUnique({
                    where: { id: BigInt(objectId) }, // objectId is guaranteed numeric by improved regex
                    select: display().label.select(),
                });
                fetchedLabelData.set(labelKey, labelDbResult ?? {});
            } catch (error) {
                logger.error(`Failed to fetch label translation for ${labelKey}`, { trace: "replaceLabels-fetch-fail", objectType, objectId, error: error instanceof Error ? error.message : String(error) });
                fetchedLabelData.set(labelKey, {}); // Cache empty object on error to avoid re-fetching and allow graceful degradation
            }
        }),
    );

    // --- Step 3: Perform replacements using pre-fetched data ---
    for (const currentUserResult of result) {
        // eslint-disable-next-line func-style
        const processAndReplaceVariables = (vars: { [key: string]: string | number } | undefined) => {
            if (!vars) return;
            for (const key of Object.keys(vars)) {
                const value = vars[key];
                if (typeof value === "string") {
                    const replacements: { placeholder: string, translatedLabel: string }[] = [];
                    let match;
                    labelRegex.lastIndex = 0; // Reset regex state for global regex before each new string value

                    while ((match = labelRegex.exec(value)) !== null) {
                        const placeholder = match[0];
                        const objectType = match[1] as ModelType;
                        const objectId = match[2];

                        const labelKey = `${objectType}:${objectId}`;
                        const currentLabelDbData = fetchedLabelData.get(labelKey);

                        // currentLabelDbData should always be found because we iterated uniqueLabelsToFetch or handled errors by setting {}
                        if (currentLabelDbData === undefined) {
                            logger.warn(`Label data unexpectedly not found in cache for ${labelKey} during replacement. Placeholder will remain.`, { trace: "replaceLabels-cache-miss-unexpected", objectType, objectId });
                            continue;
                        }

                        const { display } = ModelMap.getLogic(["display"], objectType, true, `replaceLabels value for ${objectType}:${objectId}`);
                        const translatedLabel = display().label.get(currentLabelDbData, currentUserResult.languages);
                        replacements.push({ placeholder, translatedLabel });
                    }

                    let processedValue = value;
                    for (const rep of replacements) {
                        // Replace all instances of the placeholder to ensure none remain.
                        processedValue = processedValue.replaceAll(rep.placeholder, rep.translatedLabel);
                    }
                    vars[key] = processedValue;
                }
            }
        };

        processAndReplaceVariables(currentUserResult.titleVariables);
        processAndReplaceVariables(currentUserResult.bodyVariables);
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
         * @param deliveryMode Optional delivery mode preference.
         */
        toUser: async (userId, deliveryMode) => {
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent, rawStartTs, rawEndTs } = notification;
            // Fetch recipient's preferred languages
            const userRec = await DbProvider.get().user.findUnique({
                where: { id: BigInt(userId) },
                select: { languages: true },
            });
            const recipientLanguages = userRec?.languages;
            // Shape and translate the notification for the user
            const users = await replaceLabels(
                bodyVariables,
                titleVariables,
                silent,
                [{ userId, languages: recipientLanguages, deliveryMode }],
            );
            // Override schedule/streak labels per-user
            if (rawStartTs !== undefined) {
                users[0].bodyVariables = users[0].bodyVariables ?? {};
                users[0].bodyVariables.startLabel = getEventStartLabel(
                    new Date(rawStartTs),
                    recipientLanguages && recipientLanguages.length > 0 ? recipientLanguages[0] : DEFAULT_LANGUAGE,
                );
            }
            if (rawEndTs !== undefined) {
                users[0].bodyVariables = users[0].bodyVariables ?? {};
                users[0].bodyVariables.endLabel = getEventStartLabel(
                    new Date(rawEndTs),
                    recipientLanguages && recipientLanguages.length > 0 ? recipientLanguages[0] : DEFAULT_LANGUAGE,
                );
            }
            // Send the notification
            await push({ body, bodyKey, category, link, title, titleKey, users });
        },
        /**
         * Sends a notification to multiple users
         * @param userIds The users' ids
         */
        toUsers: async (userIds, deliveryMode) => {
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent, rawStartTs, rawEndTs } = notification;
            // Prepare list of user IDs
            const ids = userIds.map(data => typeof data === "string" ? data : data.userId);
            // Fetch recipients' preferred languages
            const userRecs = await DbProvider.get().user.findMany({
                where: { id: { in: ids.map(id => BigInt(id)) } },
                select: { id: true, languages: true },
            });
            const langMap = new Map<string, string[]>();
            userRecs.forEach(u => langMap.set(u.id.toString(), u.languages));
            // Shape and translate the notification for each user
            const users = await replaceLabels(
                bodyVariables,
                titleVariables,
                silent,
                userIds.map(data => {
                    const userId = typeof data === "string" ? data : data.userId;
                    const delays = typeof data === "string" ? undefined : data.delays;
                    return { userId, delays, languages: langMap.get(userId), deliveryMode };
                }),
            );
            // Override schedule/streak labels per-user
            if (rawStartTs !== undefined) {
                for (const u of users) {
                    u.bodyVariables = u.bodyVariables ?? {};
                    const lng = u.languages && u.languages.length > 0 ? u.languages[0] : DEFAULT_LANGUAGE;
                    u.bodyVariables.startLabel = getEventStartLabel(new Date(rawStartTs), lng);
                }
            }
            if (rawEndTs !== undefined) {
                for (const u of users) {
                    u.bodyVariables = u.bodyVariables ?? {};
                    const lng = u.languages && u.languages.length > 0 ? u.languages[0] : DEFAULT_LANGUAGE;
                    u.bodyVariables.endLabel = getEventStartLabel(new Date(rawEndTs), lng);
                }
            }
            // Send the notification to each user
            await push({ body, bodyKey, category, link, title, titleKey, users });
        },
        /**
         * Sends a notification to a team
         * @param teamId The team's id
         * @param excludedUsers IDs of users to exclude from the notification
         * (usually the user who triggered the notification)
         * @param deliveryMode Optional delivery mode preference.
         */
        toTeam: async (teamId, excludedUsers, deliveryMode) => {
            const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent } = notification;
            // Find every admin of the team, excluding the user who triggered the notification
            const adminData = await DbProvider.get().member.findMany({
                where: {
                    teamId: BigInt(teamId),
                    isAdmin: true,
                    ...(typeof excludedUsers === "string" ?
                        { userId: { not: BigInt(excludedUsers) } } :
                        Array.isArray(excludedUsers) ? { userId: { notIn: excludedUsers.map(id => BigInt(id)) } } :
                            {}),
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
                deliveryMode,
            })));
            // Send the notification to each admin
            await push({ body, bodyKey, category, link, title, titleKey, users });
        },
        /**
         * Sends a notification to an owner of an object
         * @param owner The owner's id and __typename
         * @param excludedUsers IDs of users to exclude from the notification
         * @param deliveryMode Optional delivery mode preference.
         */
        toOwner: async (owner, excludedUsers, deliveryMode) => {
            if (owner.__typename === "User" && !excludedUsers?.includes(owner.id)) {
                await NotifyResult(notification).toUser(owner.id, deliveryMode);
            } else if (owner.__typename === "Team") {
                await NotifyResult(notification).toTeam(owner.id, excludedUsers, deliveryMode);
            }
        },
        /**
         * Sends a notification to all subscribers of an object
         * @param objectType The __typename of object
         * @param objectId The object's id
         * @param excludedUsers IDs of users to exclude from the notification
         * @param deliveryMode Optional delivery mode preference.
         */
        toSubscribers: async (objectType, objectId, excludedUsers, deliveryMode) => {
            try {
                // Capture default silent, rename notification-level languages
                const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent } = notification;
                const { batch } = await import("../utils/batch.js");
                await batch<Prisma.notification_subscriptionFindManyArgs, NotificationSubscriptionPayload>({
                    objectType: "NotificationSubscription",
                    select: notificationSubscriptionSelect,
                    processBatch: async (batch) => {
                        // Shape and translate the notification for each subscriber using their preferred languages
                        const users = await replaceLabels(
                            bodyVariables,
                            titleVariables,
                            silent,
                            batch.map(({ subscriberId, silent: userSpecificSilent, subscriber }) => ({
                                languages: subscriber.languages,
                                silent: userSpecificSilent ?? silent,
                                userId: subscriberId.toString(),
                                deliveryMode,
                            })),
                        );
                        // Send the notification to each subscriber
                        await push({ body, bodyKey, category, link, title, titleKey, users });
                    },
                    where: {
                        AND: [
                            { [subscribableMapper[objectType]]: { id: BigInt(objectId) } },
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
                throw error;
            }
        },
        /**
         * Sends a notification to all participants of a chat
         * @param chatId The chat's id
         * @param excludedUsers IDs of users to exclude from the notification
         * @param deliveryMode Optional deliveryMode preference.
         */
        toChatParticipants: async (chatId, excludedUsers, deliveryMode) => {
            try {
                const { body, bodyKey, bodyVariables, category, link, title, titleKey, titleVariables, silent } = notification;
                const { batch } = await import("../utils/batch.js");
                await batch<Prisma.chat_participantsFindManyArgs, ChatParticipantPayload>({
                    objectType: "ChatParticipant",
                    select: chatParticipantsSelect,
                    processBatch: async (batch) => {
                        // Shape and translate the notification for each participant
                        const users = await replaceLabels(bodyVariables, titleVariables, silent, batch.map(({ user }) => ({
                            languages: user.languages,
                            userId: user.id.toString(),
                            deliveryMode,
                        })));
                        // Send the notification to each participant
                        await push({ body, bodyKey, category, link, title, titleKey, users });
                    },
                    where: typeof excludedUsers === "string"
                        ? { AND: [{ chatId: BigInt(chatId) }, { userId: { not: BigInt(excludedUsers) } }] }
                        : Array.isArray(excludedUsers)
                            ? { AND: [{ chatId: BigInt(chatId) }, { userId: { notIn: excludedUsers.map(id => BigInt(id)) } }] }
                            : { chatId: BigInt(chatId) },
                });
            } catch (error) {
                logger.error("Caught error in toChatParticipants", { trace: "0498", error });
                throw error;
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
            deliveryMode,
        ) => {
            // If the object is subscribable, notify subscribers
            const isSubscribable = isObjectSubscribable(objectType);
            if (isSubscribable) {
                await NotifyResult(notification).toSubscribers(objectType as string as SubscribableObject, objectId, excludedUsers, deliveryMode);
            }
            // Notify the owner
            if (owner) {
                await NotifyResult(notification).toOwner(owner, excludedUsers, deliveryMode);
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
                            id: generatePK(),
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
        testPushDevice: async (deviceId: string, userId: string): Promise<Success> => {
            // Verify the push device belongs to the user
            const device = await DbProvider.get().push_device.findUnique({
                where: { id: BigInt(deviceId) },
                select: { endpoint: true, p256dh: true, auth: true, userId: true },
            });
            if (!device || device.userId.toString() !== userId) {
                throw new CustomError("0308", "PushDeviceNotFound");
            }
            const { endpoint, p256dh, auth: deviceAuth } = device;
            // Send the test notification using the stored credentials
            return QueueService.get().push.addTask({
                type: QueueTaskType.PUSH_NOTIFICATION,
                id: nanoid(),
                subscription: { endpoint, keys: { p256dh, auth: deviceAuth } },
                payload: { body: "This is a test notification", icon: APP_ICON, link: "https://vrooli.com/", title: "Testing!" },
            });
        },
        /**
         * Updates a user's notification settings
         * @param settings The new settings
         */
        updateSettings: async (settings: NotificationSettingsUpdateInput, userId: string) => {
            return await updateNotificationSettings(settings, userId);
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
            bodyVariables: {
                title: `<Label|${scheduleForType}:${scheduleForId}>`,
            },
            rawStartTs: startTime.getTime(),
            category: "Schedule",
            languages,
            link: scheduleForType === "Meeting" ? `/meeting/${scheduleForId}` : undefined,
        }),
        pushStreakReminder: (timeToReset: Date): NotifyResultType => NotifyResult({
            bodyKey: "StreakReminder_Body",
            rawEndTs: timeToReset.getTime(),
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
            bodyKey: "TransferAccepted_Body",
            bodyVariables: { objectName: `<Label|${objectType}:${objectId}>` },
            category: "Transfer",
            languages,
            link: `/${LINKS[objectType]}/${objectId}`,
            titleKey: "TransferAccepted_Title",
        }),
        pushTransferRejected: (objectType: `${ModelType}`, objectId: string): NotifyResultType => NotifyResult({
            bodyKey: "TransferRejected_Body",
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
        /**
         * Notification for when a tool call requires user approval.
         */
        pushToolApprovalRequired: (
            conversationId: string,
            pendingId: string,
            toolName: string,
            callerBotName?: string,
            conversationName?: string,
            approvalTimeoutAt?: number, // Epoch ms
            estimatedCost?: string | number, // Stringified bigint or number
            autoRejectOnTimeout?: boolean,
        ): NotifyResultType => {
            let bodyKey: TransKey = "ToolApprovalRequired_Body_Free_Reject"; // Default case
            const hasCost = estimatedCost && BigInt(estimatedCost) > 0;
            const effectiveAutoRejectOnTimeout = autoRejectOnTimeout ?? true; // Default to reject on timeout

            if (hasCost) {
                bodyKey = effectiveAutoRejectOnTimeout ? "ToolApprovalRequired_Body_Cost_Reject" : "ToolApprovalRequired_Body_Cost_Proceed";
            } else {
                bodyKey = effectiveAutoRejectOnTimeout ? "ToolApprovalRequired_Body_Free_Reject" : "ToolApprovalRequired_Body_Free_Proceed";
            }

            const bodyVariables: { [key: string]: string | number } = {
                toolName,
                callerBotName: callerBotName || "A bot",
                conversationName: conversationName || "your Vrooli task",
            };

            if (hasCost) {
                bodyVariables.estimatedCost = typeof estimatedCost === "string" ? estimatedCost : estimatedCost!.toString();
            }

            // rawEndTs is used by NotifyResult -> replaceLabels -> getEventStartLabel to generate 'timeRemaining'
            // We use rawEndTs because getEventStartLabel calculates "in X time" based on the current time when it's called.
            let rawTimeoutTs: number | undefined;
            if (approvalTimeoutAt && approvalTimeoutAt > Date.now()) {
                rawTimeoutTs = approvalTimeoutAt;
                // The actual 'timeRemaining' variable will be populated by getEventStartLabel
                // when i18next processes the key in the toUser/toUsers methods, using the rawEndTs.
                // For the {{#if timeRemaining}} conditional in the translation, we just need *a* value for timeRemaining.
                // So, we pass a placeholder here, and getEventStartLabel will compute the actual human-readable string.
                bodyVariables.timeRemaining = " "; // Placeholder to ensure conditional block is entered
            }

            return NotifyResult({
                bodyKey,
                bodyVariables,
                category: "Run", // Consider a new category "ToolApproval" or similar in the future
                languages,
                link: `${endpointsTask.respondToToolApproval.endpoint}?conversationId=${conversationId}&pendingId=${pendingId}`,
                titleKey: "ToolApprovalRequired_Title",
                rawEndTs: rawTimeoutTs, // Pass the timeout for timeRemaining calculation
            });
        },
        /**
         * Notification for when a long-running task has been running for too long and will be paused/stopped.
         */
        pushLongRunningTaskWarning: (
            taskId: string,
            taskType: string,
            durationMs: number,
            thresholdMs: number,
        ): NotifyResultType => {
            const durationSeconds = Math.floor(durationMs / SECONDS_1_MS).toString();
            const thresholdSeconds = Math.floor(thresholdMs / SECONDS_1_MS).toString();

            const bodyVariables = {
                taskType,
                taskId,
                duration: durationSeconds,
                threshold: thresholdSeconds,
            };

            return NotifyResult({
                category: "SystemAlert" as NotificationCategory,
                titleKey: "LongRunningTaskWarning_Title",
                bodyKey: "LongRunningTaskWarning_Body",
                bodyVariables,
                link: `/tasks/${taskId}`,
                languages,
            });
        },
    };
}
