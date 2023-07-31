import { calculateOccurrences, GqlModelType, uppercaseFirstLetter } from "@local/shared";
import { Prisma } from "@prisma/client";
import { findFirstRel } from "../../builders";
import { logger, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe, schedulesWhereInTimeframe } from "../../events";
import { Notify, parseSubscriptionContext, ScheduleSubscriptionContext } from "../../notify";
import { withRedis } from "../../redisConn";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { batchCollect } from "../../utils/batchCollect";

/**
 * For a list of scheduled events, finds subscribers and schedules notifications for them
 * @param prisma The Prisma client
 * @param scheduleId The ID of the schedule to find subscribers for
 * @param occurrences Windows of time that the schedule will occur. Subscribers will receive 
 * one or more push notifications before each occurrence, depending on their notification preferences.
 */
const scheduleNotifications = async (
    prisma: PrismaType,
    scheduleId: string,
    occurrences: { start: Date, end: Date }[],
) => {
    await withRedis({
        process: async (redisClient) => {
            await batchCollect<Prisma.notification_subscriptionFindManyArgs>({
                objectType: "NotificationSubscription",
                prisma,
                processBatch: async (batch) => {
                    // If no subscriptions, continue
                    if (batch.length === 0) return;
                    // Find objectType and id of the object that has a schedule. Assume there is only one focusMode, meeting, etc.
                    const [objectField, objectData] = findFirstRel(batch[0].schedule!, ["focusModes", "meetings", "runProjects", "runRoutines"]);
                    if (!objectField || !objectData || objectData.length === 0) {
                        logger.error(`Could not find object type for schedule ${scheduleId}`, { trace: "0433" });
                        return;
                    }
                    const scheduleForId = objectData[0].id;
                    const scheduleForType = uppercaseFirstLetter(objectField.slice(0, -1)) as GqlModelType;
                    // Find notification preferences for each subscriber
                    const subscriberPrefs: { [userId: string]: ScheduleSubscriptionContext["reminders"] } = {};
                    for (const subscription of batch) {
                        const context = parseSubscriptionContext(subscription.context) as ScheduleSubscriptionContext | null;
                        if (!context) continue;
                        for (const reminder of context.reminders) {
                            if (!subscriberPrefs[subscription.subscriber.id]) {
                                subscriberPrefs[subscription.subscriber.id] = [];
                            }
                            subscriberPrefs[subscription.subscriber.id].push(reminder);
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
                        const redisGetResults = await Promise.all(redisKeys.map(key => redisClient.get(key)));
                        // Filter out subscribers who have already been notified
                        const filteredSubscriberDelaysList = subscriberDelaysList.filter((_, index) => !redisGetResults[index]);
                        // Send push notifications to each subscriber
                        await Notify(prisma, ["en"]).pushScheduleReminder(scheduleForId, scheduleForType, occurrence.start).toUsers(filteredSubscriberDelaysList);

                        // Set Redis keys for the subscribers who just received the notification, with an expiration time of 25 hours
                        await Promise.all(filteredSubscriberDelaysList.map((subscriber) => {
                            const key = `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`;
                            return redisClient.set(key, "true", { EX: 25 * 60 * 60 });
                        }));
                    }
                },
                select: {
                    id: true,
                    context: true,
                    silent: true,
                    schedule: {
                        select: {
                            id: true,
                            focusModes: { select: { id: true } },
                            meetings: { select: { id: true } },
                            runProjects: { select: { id: true } },
                            runRoutines: { select: { id: true } },
                        },
                    },
                    subscriber: {
                        select: { id: true },
                    },
                },
                where: {
                    schedule: { id: scheduleId },
                },
            });
        },
        trace: "0511",
    });
};

/**
 * Caches upcoming scheduled events in the database.
 */
export const scheduleNotify = async () => {
    // Define window for start and end dates. 
    // Should be looking for all events that occur within the next 25 hours. 
    // This script runs every 24 hours, so we add an hour buffer in case the script runs late.
    const now = new Date();
    const startDate = now;
    const endDate = new Date(now.setHours(now.getHours() + 25));
    await batch<Prisma.scheduleFindManyArgs>({
        objectType: "Schedule",
        processBatch: async (batch, prisma) => {
            Promise.all(batch.map(async (schedule) => {
                // Find all occurrences of the schedule within the next 25 hours
                const occurrences = calculateOccurrences(schedule, startDate, endDate);
                // For each occurrence, schedule notifications for subscribers of the schedule
                await scheduleNotifications(prisma, schedule.id, occurrences);
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
        trace: "0428",
        where: schedulesWhereInTimeframe(startDate, endDate),
    });
};
