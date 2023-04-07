import Bull from 'bull';
import { HOST, PORT } from '../../redisConn.js';
import { pushProcess } from './process.js';

export type PushSubscription = {
    endpoint: string;
    keys: {
        p256dh: string;
        auth: string;
    };
}

export type PushPayload = {
    body: string;
    icon?: string | null;
    link: string | null | undefined;
    title: string | null | undefined;
}

const pushQueue = new Bull('push', { redis: { port: PORT, host: HOST } });
pushQueue.process(pushProcess as any);

export function sendPush(subscription: PushSubscription, payload: PushPayload, delay: number = 0) {
    pushQueue.add({
        ...payload,
        ...subscription,
    }, { delay });
}