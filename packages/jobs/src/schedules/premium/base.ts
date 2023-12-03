/**
 * Removes expired premium status from users. This might already be handled by Stripe, but this is
 * a backup in case something goes wrong.
 * 
 * A cron job is triggered to run this every day at 4:20am (UTC).
 */
import { cronTimes } from "../../cronTimes";
import { initializeCronJob } from "../../initializeCronJob";
import { expirePremium } from "./expirePremium";
import { failPayments } from "./failPayments";

/** Initializes cron jobs for expiring premium status */
export const initExpirePremiumCronJob = () => {
    initializeCronJob(cronTimes.expirePremium, expirePremium, "expire premium");
};

/** Initializes cron jobs for marking payments stuck in pending to failed */
export const initFailPaymentsCronJob = () => {
    initializeCronJob(cronTimes.failPayments, failPayments, "fail payments");
};
