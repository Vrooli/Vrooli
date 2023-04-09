import { logger } from "../../events";
import webpush from 'web-push';
import { PushPayload, PushSubscription } from "./queue";

webpush.setVapidDetails(
    `mailto:${process.env.LETSENCRYPT_EMAIL}`,
    process.env.VAPID_PUBLIC_KEY ?? '',
    process.env.VAPID_PRIVATE_KEY ?? ''
);

export async function pushProcess(job: PushSubscription & PushPayload) {
    try {
        const subscription = {
            endpoint: job.endpoint,
            keys: {
                p256dh: job.keys.p256dh,
                auth: job.keys.auth,
            },
        }
        await webpush.sendNotification(subscription, JSON.stringify({
            body: job.body,
            icon: job.icon,
            link: job.link,
            title: job.title,
        }));
    } catch (err) {
        logger.error('Error sending push notification', { trace: '0308' });
    }
}