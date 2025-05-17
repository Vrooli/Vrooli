import { Job } from "bullmq";
import webpush from "web-push";
import { logger } from "../../events/logger.js";
import { PushNotificationTask } from "../taskTypes.js";

let vapidDetailsSet = false;

/**
 * Function to set VAPID details. This is needed because 
 * the private key is loaded from the secrets location, so it's 
 * not available at startup.
 */
export function setVapidDetails() {
    if (!vapidDetailsSet) {
        webpush.setVapidDetails(
            `mailto:${process.env.LETSENCRYPT_EMAIL}`,
            process.env.VAPID_PUBLIC_KEY ?? "",
            process.env.VAPID_PRIVATE_KEY ?? "",
        );
        vapidDetailsSet = true;
    }
}

export async function pushProcess({ data }: Job<PushNotificationTask>) {
    try {
        const { subscription, payload } = data;
        setVapidDetails();
        await webpush.sendNotification(subscription, JSON.stringify(payload));
    } catch (err) {
        logger.error("Error sending push notification", { trace: "0308" });
    }
}
