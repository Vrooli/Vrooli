import { Job } from "bull";
import webpush from "web-push";
import { logger } from "../../events/logger";
import { PushPayload, PushSubscription } from "./queue";

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

export async function pushProcess({ data }: Job<PushSubscription & PushPayload>) {
    try {
        const { endpoint, keys, body, icon, link, title } = data;
        setVapidDetails();
        const subscription = {
            endpoint,
            keys: {
                p256dh: keys.p256dh,
                auth: keys.auth,
            },
        };
        await webpush.sendNotification(subscription, JSON.stringify({ body, icon, link, title }));
    } catch (err) {
        logger.error("Error sending push notification", { trace: "0308" });
    }
}
