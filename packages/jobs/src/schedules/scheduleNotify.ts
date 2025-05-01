import { Notify, ScheduleSubscriptionContext, batch, findFirstRel, logger, parseJsonOrDefault, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe, schedulesWhereInTimeframe, withRedis } from "@local/server";
import { ModelType, calculateOccurrences, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";

/**
 * For a list of scheduled events, finds subscribers and schedules notifications for them
 * @param scheduleId The ID of the schedule to find subscribers for
 * @param occurrences Windows of time that the schedule will occur. Subscribers will receive 
 * one or more push notifications before each occurrence, depending on their notification preferences.
 */
async function scheduleNotifications(
    scheduleId: string,
    occurrences: { start: Date, end: Date }[],
) {
    await withRedis({
        process: async (redisClient) => {
            await batch<Prisma.notification_subscriptionFindManyArgs>({
                objectType: "NotificationSubscription",
                processBatch: async (batch) => {
                    // If no subscriptions, continue
                    if (batch.length === 0) return;
                    // Find objectType and id of the object that has a schedule. Assume there is only one runRoutine, meeting, etc.
                    const [objectField, objectData] = findFirstRel(batch[0].schedule!, ["meetings", "runs"]);
                    if (!objectField || !objectData || objectData.length === 0) {
                        logger.error(`Could not find object type for schedule ${scheduleId}`, { trace: "0433" });
                        return;
                    }
                    const scheduleForId: string = objectData[0].id.toString();
                    const scheduleForType = uppercaseFirstLetter(objectField.slice(0, -1)) as ModelType;
                    // Find notification preferences for each subscriber
                    const subscriberPrefs: { [userId: string]: ScheduleSubscriptionContext["reminders"] } = {};
                    for (const subscription of batch) {
                        const subscriberId = subscription.subscriber.id.toString();
                        const context = parseJsonOrDefault<ScheduleSubscriptionContext>(subscription.context, {} as ScheduleSubscriptionContext);
                        if (!context) continue;
                        for (const reminder of context.reminders) {
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
                        // Reminder has "minutesBefore" property which specifies how many minutes before each occurrence to send the notification. 
                        // We must convert this to a delay from now, using the formula: 
                        // delay = eventStart - (minutesBefore * 60 * 1000) - now
                        for (const userId in subscriberPrefs) {
                            for (const reminder of subscriberPrefs[userId]) {
                                const delay = occurrence.start.getTime() - (reminder.minutesBefore * 60 * 1000) - Date.now();
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
                        const redisGetResults = redisClient ? await Promise.all(redisKeys.map(key => redisClient.get(key))) : null;
                        // Filter out subscribers who have already been notified
                        const filteredSubscriberDelaysList = redisGetResults ? subscriberDelaysList.filter((_, index) => !redisGetResults[index]) : subscriberDelaysList;
                        // Send push notifications to each subscriber
                        //TODO should add to bull queue to notify at correct time
                        await Notify(["en"]).pushScheduleReminder(scheduleForId, scheduleForType, occurrence.start).toUsers(filteredSubscriberDelaysList);

                        // Set Redis keys for the subscribers who just received the notification, with an expiration time of 25 hours
                        if (redisClient) {
                            await Promise.all(filteredSubscriberDelaysList.map((subscriber) => {
                                const key = `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`;
                                return redisClient.set(key, "true", { EX: 25 * 60 * 60 });
                            }));
                        }
                    }
                },
                select: {
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
                },
                where: {
                    schedule: { id: BigInt(scheduleId) },
                },
            });
        },
        trace: "0511",
    });
}

/**
 * Caches upcoming scheduled events in the database.
 */
export async function scheduleNotify() {
    try {
        // Define window for start and end dates. 
        // Should be looking for all events that occur within the next 25 hours. 
        // This script runs every 24 hours, so we add an hour buffer in case the script runs late.
        const now = new Date();
        const startDate = now;
        const endDate = new Date(now.setHours(now.getHours() + 25));
        await batch<Prisma.scheduleFindManyArgs>({
            objectType: "Schedule",
            processBatch: async (batch) => {
                Promise.all(batch.map(async (schedule) => {
                    const scheduleId = schedule.id.toString();
                    // Find all occurrences of the schedule within the next 25 hours
                    const occurrences = await calculateOccurrences(schedule, startDate, endDate);
                    // For each occurrence, schedule notifications for subscribers of the schedule
                    await scheduleNotifications(scheduleId, occurrences);
                }));
            },
            select: {
                id: true,
                startTime: true,
                endTime: true,
                timezone: true,
                exceptions: {
                    where: scheduleExceptionsWhereInTimeframe(startDate, endDate),
                    select: {
                        id: true,
                        originalStartTime: true,
                        newStartTime: true,
                        newEndTime: true,
                    },
                },
                recurrences: {
                    where: scheduleRecurrencesWhereInTimeframe(startDate, endDate),
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
            },
            where: schedulesWhereInTimeframe(startDate, endDate),
        });
    } catch (error) {
        logger.error("scheduleNotify caught error", { error, trace: "0428" });
    }
}
