import { calculateOccurrences, uppercaseFirstLetter } from "@local/utils";
import { PrismaClient } from "@prisma/client";
import { findFirstRel } from "../../builders";
import { logger, scheduleExceptionsWhereInTimeframe, scheduleRecurrencesWhereInTimeframe, schedulesWhereInTimeframe } from "../../events";
import { Notify, parseSubscriptionContext } from "../../notify";
import { initializeRedis } from "../../redisConn";
const scheduleNotifications = async (prisma, scheduleId, occurrences) => {
    const redisClient = await initializeRedis();
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.notification_subscription.findMany({
            where: {
                schedule: { id: scheduleId },
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
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        if (batch.length === 0)
            continue;
        const [objectField, objectData] = findFirstRel(batch[0].schedule, ["focusModes", "meetings", "runProjects", "runRoutines"]);
        if (!objectField || !objectData || objectData.length === 0) {
            logger.error(`Could not find object type for schedule ${scheduleId}`, { trace: "0433" });
            continue;
        }
        const scheduleForId = objectData[0].id;
        const scheduleForType = uppercaseFirstLetter(objectField.slice(0, -1));
        const subscriberPrefs = {};
        for (const subscription of batch) {
            const context = parseSubscriptionContext(subscription.context);
            if (!context)
                continue;
            for (const reminder of context.reminders) {
                if (!subscriberPrefs[subscription.subscriber.id]) {
                    subscriberPrefs[subscription.subscriber.id] = [];
                }
                subscriberPrefs[subscription.subscriber.id].push(reminder);
            }
        }
        for (const occurrence of occurrences) {
            const subscriberDelays = {};
            for (const userId in subscriberPrefs) {
                for (const reminder of subscriberPrefs[userId]) {
                    const delay = occurrence.start.getTime() - (reminder.minutesBefore * 60 * 1000) - Date.now();
                    if (!subscriberDelays[userId]) {
                        subscriberDelays[userId] = [];
                    }
                    subscriberDelays[userId].push(delay);
                }
            }
            const subscriberDelaysList = [];
            for (const userId in subscriberDelays) {
                subscriberDelaysList.push({
                    userId,
                    delays: subscriberDelays[userId],
                });
            }
            const redisKeys = subscriberDelaysList.map(subscriber => `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`);
            const redisGetResults = await Promise.all(redisKeys.map(key => redisClient.get(key)));
            const filteredSubscriberDelaysList = subscriberDelaysList.filter((_, index) => !redisGetResults[index]);
            await Notify(prisma, ["en"]).pushScheduleReminder(scheduleForId, scheduleForType, occurrence.start).toUsers(filteredSubscriberDelaysList);
            await Promise.all(filteredSubscriberDelaysList.map((subscriber) => {
                const key = `notify-schedule:${scheduleId}:${occurrence.start.getTime()}:user:${subscriber.userId}`;
                return redisClient.set(key, "true", { EX: 25 * 60 * 60 });
            }));
        }
    } while (currentBatchSize === batchSize);
};
export const scheduleNotify = async () => {
    const prisma = new PrismaClient();
    try {
        const now = new Date();
        const startDate = now;
        const endDate = new Date(now.setHours(now.getHours() + 25));
        const batchSize = 100;
        const skip = 0;
        let currentBatchSize = 0;
        do {
            const schedules = await prisma.schedule.findMany({
                where: schedulesWhereInTimeframe(startDate, endDate),
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
                skip,
                take: batchSize,
            });
            currentBatchSize = schedules.length;
            for (const schedule of schedules) {
                const occurrences = calculateOccurrences(schedule, startDate, endDate);
                await scheduleNotifications(prisma, schedule.id, occurrences);
            }
        } while (currentBatchSize === batchSize);
        logger.info("Upcoming scheduled events cached successfully.", { trace: "0427" });
    }
    catch (error) {
        logger.error("Failed to cache upcoming scheduled events.", { trace: "0428" });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=scheduleNotify.js.map