/**
 * Handles sending push notifications for upcoming scheduled events in the database
 */
import { cronTimes } from "../../cronTimes";
import { initializeCronJob } from "../../initializeCronJob";
import { scheduleNotify } from "./scheduleNotify";

/**
 * Initializes cron jobs for schedule event notifications
 */
export const initEventsCronJobs = () => {
    initializeCronJob(cronTimes.events, scheduleNotify, "schedule events");
};
