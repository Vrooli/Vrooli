import { type Prisma } from "@prisma/client";
import { CacheService, Notify, batch, findFirstRel, logger, parseJsonOrDefault, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe, schedulesWhereInTimeframe, type ScheduleSubscriptionContext } from "@vrooli/server";
import { calculateOccurrences, uppercaseFirstLetter, ModelType, type ModelType as ModelTypeType, type Schedule } from "@vrooli/shared";

const MINUTES_IN_HOUR = 60;
const SECONDS_IN_MINUTE = 60;
const MILLISECONDS_IN_SECOND = 1000;
const BUFFER_HOURS = 25;

/**
 * Type guard to check if a string is a valid ModelType
 */
function isValidModelType(value: string): value is ModelTypeType {
    return (Object.values(ModelType) as string[]).includes(value);
}

// Define select shape and payload type for notification subscriptions
const notificationSubscriptionSelect = {
    id: true,
    context: true,
    silent: true,
    schedule: {
        select: {
            id: true,
            meetings: { select: { id: true } },
            runs: { select: { id: true } },
        },
    },
    subscriber: {
        select: { id: true },
    },
} as const;
type NotificationSubscriptionPayload = Prisma.notification_subscriptionGetPayload<{ select: typeof notificationSubscriptionSelect }>;

// Define select shape and payload type for schedules
// Note: The `where` clauses for exceptions and recurrences cannot be part of the `select` constant
// as they depend on `startDate` and `endDate` which are defined within the `scheduleNotify` function.
// These will be applied directly in the `batch` call's `select` argument.
const scheduleSelect = {
    id: true,
    startTime: true,
    endTime: true,
    timezone: true,
    createdAt: true,
    publicId: true,
    updatedAt: true,
    meetings: { select: { id: true } },
    runs: { select: { id: true } },
    exceptions: {
        // `where` will be added dynamically
        select: {
            id: true,
            originalStartTime: true,
            newStartTime: true,
            newEndTime: true,
        },
    },
    recurrences: {
        // `where` will be added dynamically
        select: {
            id: true,
            recurrenceType: true,
            interval: true,
            dayOfWeek: true,
            dayOfMonth: true,
            month: true,
            endDate: true,
        },
    },
} as const;
type SchedulePayload = Prisma.scheduleGetPayload<{ select: typeof scheduleSelect }>;

/**
 * For a list of scheduled events, finds subscribers and schedules notifications for them
 * @param scheduleId The ID of the schedule to find subscribers for
 * @param occurrences Windows of time that the schedule will occur. Subscribers will receive 
 * one or more push notifications before each occurrence, depending on their notification preferences.
 */
async function scheduleNotifications(
    scheduleId: string,
    occurrences: { start: Date, end: Date }[],
): Promise<void> {
    try {
        const cache = CacheService.get();
        await batch<Prisma.notification_subscriptionFindManyArgs, NotificationSubscriptionPayload>({
            objectType: "NotificationSubscription",
            processBatch: async (batchResult) => {
                // If no subscriptions, continue
                if (batchResult.length === 0) return;

                // Safely access the schedule from the first batch result
                const firstSubscription = batchResult.length > 0 ? batchResult[0] : null;
                if (!firstSubscription || !firstSubscription.schedule) {
                    // It's possible scheduleId is not defined if batchResult is empty, but we already checked that.
                    // However, firstSubscription.schedule could still be null/undefined if the query somehow allows it
                    // or if there's an issue with data integrity or query hydration.
                    logger.error(`Subscription batch for schedule ${scheduleId} has missing schedule data in the first item. Batch processing aborted for this set.`, {
                        scheduleId,
                        firstSubscriptionId: firstSubscription?.id, // Log ID if available
                        trace: "SN_MISSING_SCHED_DATA", // New trace ID for this specific error
                    });
                    return; // Abort processing for this batch if critical schedule info is missing
                }

                // Find objectType and id of the object that has a schedule. Assume there is only one runRoutine, meeting, etc.
                const [objectField, objectData] = findFirstRel(firstSubscription.schedule, ["meetings", "runs"]);
                if (!objectField || !objectData || objectData.length === 0) {
                    logger.error(`Could not find object type for schedule ${scheduleId}`, { trace: "0433" });
                    return;
                }
                // Safe array access with validation
                const firstObject = objectData && Array.isArray(objectData) && objectData.length > 0 ? objectData[0] : null;
                if (!firstObject || !firstObject.id) {
                    logger.error(`No valid object found in ${objectField} for schedule ${scheduleId}`, { trace: "0433_no_object" });
                    return;
                }
                const scheduleForId: string = firstObject.id.toString();
                const scheduleForType = uppercaseFirstLetter(objectField.slice(0, -1));
                
                // Validate that scheduleForType is a valid ModelType
                if (!isValidModelType(scheduleForType)) {
                    logger.error(`Invalid ModelType derived from objectField: ${objectField}`, {
                        objectField,
                        derivedType: scheduleForType,
                        validTypes: Object.values(ModelType),
                        trace: "0433_invalid_type",
                    });
                    return;
                }
                // TypeScript now knows scheduleForType is ModelTypeType after the type guard
                // Find notification preferences for each subscriber
                const subscriberPrefs: { [userId: string]: ScheduleSubscriptionContext["reminders"] } = {};
                for (const subscription of batchResult) {
                    // Issue 3 fix: Add null check for subscriber and subscriber.id
                    if (!subscription.subscriber?.id) {
                        logger.warn(`Subscription ${subscription.id} is missing subscriber or subscriber ID. Skipping this subscription.`, {
                            subscriptionId: subscription.id,
                            subscriptionContext: subscription.context,
                            hasSubscriber: !!subscription.subscriber,
                            trace: "SN_MISSING_SUB_ID",
                        });
                        continue;
                    }
                    const subscriberId = subscription.subscriber.id.toString();

                    // Issue 2 fix: Provide a safer default for parseJsonOrDefault
                    const context = parseJsonOrDefault<ScheduleSubscriptionContext>(
                        subscription.context,
                        { reminders: [] }, // Default to an object with an empty reminders array
                    );

                    // Validate the parsed context matches expected structure
                    if (!context || typeof context !== "object") {
                        logger.warn(`Subscription ${subscription.id} for subscriber ${subscriberId} has invalid context structure after parsing. Skipping reminder processing.`, {
                            subscriptionId: subscription.id,
                            subscriberId,
                            parsedContext: context,
                            contextType: typeof context,
                            trace: "SN_CTX_INVALID_TYPE",
                        });
                        continue;
                    }

                    // Ensure context and context.reminders are valid before proceeding
                    if (!Array.isArray(context.reminders)) {
                        logger.warn(`Subscription ${subscription.id} for subscriber ${subscriberId} has context without a valid 'reminders' array after parsing. Skipping reminder processing.`, {
                            subscriptionId: subscription.id,
                            subscriberId,
                            parsedContext: context,
                            remindersType: typeof context.reminders,
                            trace: "SN_CTX_NO_REMINDERS",
                        });
                        continue;
                    }

                    for (const reminder of context.reminders) {
                        // Optional: Add further validation for the reminder object itself if its structure is critical
                        if (typeof reminder.minutesBefore !== "number") {
                            logger.warn(`Subscription ${subscription.id} for subscriber ${subscriberId} has a reminder with invalid 'minutesBefore' type or value. Skipping this reminder.`, {
                                subscriptionId: subscription.id,
                                subscriberId,
                                reminder,
                                trace: "SN_INV_REM_MIN", // New trace ID
                            });
                            continue; // Skip this specific reminder
                        }

                        if (!subscriberPrefs[subscriberId]) {
                            subscriberPrefs[subscriberId] = [];
                        }
                        subscriberPrefs[subscriberId].push(reminder);
                    }
                }
                // For each occurrence
                for (const occurrence of occurrences) {
                    // Find all delays for notifications
                    const subscriberDelays: { [userId: string]: number[] } = {};
                    const now = Date.now(); // Cache Date.now()
                    // Reminder has "minutesBefore" property which specifies how many minutes before each occurrence to send the notification. 
                    // We must convert this to a delay from now, using the formula: 
                    // delay = eventStart - (minutesBefore * 60 * 1000) - now
                    for (const userId in subscriberPrefs) {
                        for (const reminder of subscriberPrefs[userId]) {
                            const delay = occurrence.start.getTime() - (reminder.minutesBefore * MINUTES_IN_HOUR * SECONDS_IN_MINUTE * MILLISECONDS_IN_SECOND) - now; // Use cached now
                            if (!subscriberDelays[userId]) {
                                subscriberDelays[userId] = [];
                            }
                            subscriberDelays[userId].push(delay);
                        }
                    }
                    // Convert arrays to list
                    const subscriberDelaysList: { userId: string, delays: number[] }[] = [];
                    for (const userId in subscriberDelays) {
                        subscriberDelaysList.push({
                            userId,
                            delays: subscriberDelays[userId],
                        });
                    }
                    // Before sending notifications, check Redis to see if the user has already been notified for this event
                    const redisKeys = subscriberDelaysList.map(subscriber => `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`);
                    // Assuming CacheService.get() might return null or throw if key not found.
                    // And that CacheService.get() can take an array of keys.
                    // For simplicity, fetching one by one. If CacheService supports batch get, that would be better.
                    const redisGetResultsPromises = redisKeys.map(key => cache.get<string>(key));
                    const redisGetResults = await Promise.all(redisGetResultsPromises);

                    // Filter out subscribers who have already been notified
                    const filteredSubscriberDelaysList = subscriberDelaysList.filter((_, index) => !redisGetResults[index]);
                    // Send push notifications to each subscriber
                    //TODO should add to bull queue to notify at correct time
                    await Notify(["en"]).pushScheduleReminder(scheduleForId, scheduleForType, occurrence.start).toUsers(filteredSubscriberDelaysList);

                    // Set Redis keys for the subscribers who just received the notification, with an expiration time of 25 hours
                    const ttlSeconds = BUFFER_HOURS * MINUTES_IN_HOUR * SECONDS_IN_MINUTE;
                    await Promise.all(filteredSubscriberDelaysList.map((subscriber) => {
                        const key = `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`;
                        return cache.set(key, "true", ttlSeconds);
                    }));
                }
            },
            select: notificationSubscriptionSelect,
            where: {
                schedule: { id: BigInt(scheduleId) },
            },
        });
    } catch (error) {
        logger.error("scheduleNotifications caught error during CacheService interaction or batch processing", { error, trace: "0511_cache_service" });
    }
}

/**
 * Caches upcoming scheduled events in the database.
 */
export async function scheduleNotify(): Promise<void> {
    try {
        // Define window for start and end dates. 
        // Should be looking for all events that occur within the next 25 hours. 
        // This script runs every 24 hours, so we add an hour buffer in case the script runs late.
        const now = new Date();
        const startDate = now;
        const endDate = new Date(new Date().setHours(now.getHours() + BUFFER_HOURS));
        await batch<Prisma.scheduleFindManyArgs, SchedulePayload>({
            objectType: "Schedule",
            processBatch: async (batchResult) => {
                await Promise.all(batchResult.map(async (scheduleData) => {
                    const scheduleIdStr = scheduleData.id.toString();

                    // Validate and map exceptions with proper type safety
                    const exceptions: Schedule["exceptions"] = scheduleData.exceptions
                        .filter(ex => ex && typeof ex.id !== "undefined" && ex.originalStartTime)
                        .map(ex => ({
                            id: ex.id.toString(),
                            __typename: "ScheduleException" as const,
                            originalStartTime: new Date(ex.originalStartTime),
                            newStartTime: ex.newStartTime ? new Date(ex.newStartTime) : null,
                            newEndTime: ex.newEndTime ? new Date(ex.newEndTime) : null,
                        }));

                    // Validate and map recurrences with proper type safety
                    const recurrences: Schedule["recurrences"] = scheduleData.recurrences
                        .filter(rec => rec && typeof rec.id !== "undefined" && rec.recurrenceType)
                        .map(rec => ({
                            id: rec.id.toString(),
                            __typename: "ScheduleRecurrence" as const,
                            recurrenceType: rec.recurrenceType,
                            interval: rec.interval,
                            dayOfWeek: rec.dayOfWeek,
                            dayOfMonth: rec.dayOfMonth,
                            month: rec.month,
                            endDate: rec.endDate ? new Date(rec.endDate) : null,
                            // duration: rec.duration, // Only include if 'duration' is in ScheduleRecurrencePayload and ScheduleRecurrence type
                        }));

                    // Find all occurrences of the schedule within the next 25 hours
                    const occurrences = await calculateOccurrences({
                        startTime: new Date(scheduleData.startTime),
                        endTime: new Date(scheduleData.endTime),
                        exceptions,
                        recurrences,
                        timezone: scheduleData.timezone,
                    }, startDate, endDate);
                    // For each occurrence, schedule notifications for subscribers of the schedule
                    await scheduleNotifications(scheduleIdStr, occurrences);
                }));
            },
            select: {
                ...scheduleSelect,
                exceptions: {
                    where: scheduleExceptionsWhereInTimeframe(startDate, endDate),
                    select: scheduleSelect.exceptions.select,
                },
                recurrences: {
                    where: scheduleRecurrencesWhereInTimeframe(startDate, endDate),
                    select: scheduleSelect.recurrences.select,
                },
            },
            where: schedulesWhereInTimeframe(startDate, endDate),
        });
    } catch (error) {
        logger.error("scheduleNotify caught error", { error, trace: "0428" });
    }
}
