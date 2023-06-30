import webpush from "web-push";
import { logger } from "../../events";
import { PushPayload, PushSubscription } from "./queue";

let vapidDetailsSet = false;

/**
 * Function to set VAPID details. This is needed because 
 * the private key is loaded from the secrets location, so it's 
 * not available at startup.
 */
export const setVapidDetails = () => {
    if (!vapidDetailsSet) {
        webpush.setVapidDetails(
            `mailto:${process.env.LETSENCRYPT_EMAIL}`,
            process.env.VAPID_PUBLIC_KEY ?? "",
            process.env.VAPID_PRIVATE_KEY ?? "",
        );
        vapidDetailsSet = true;
    }
};

export async function pushProcess(job: PushSubscription & PushPayload) {
    try {
        setVapidDetails();
        const subscription = {
            endpoint: job.endpoint,
            keys: {
                p256dh: job.keys.p256dh,
                auth: job.keys.auth,
            },
        };
        await webpush.sendNotification(subscription, JSON.stringify({
            body: job.body,
            icon: job.icon,
            link: job.link,
            title: job.title,
        }));
    } catch (err) {
        logger.error("Error sending push notification", { trace: "0308" });
    }
}
