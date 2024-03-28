import Bull from "bull";
import winston from "winston";

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

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let pushProcess: (job: Bull.Job<PushSubscription & PushPayload>) => Promise<unknown>;
let pushQueue: Bull.Queue<PushSubscription & PushPayload>;

// Call this on server startup
export async function setupPushQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        pushProcess = processModule.pushProcess;

        // Initialize the Bull queue
        pushQueue = new Bull<PushSubscription & PushPayload>("push", {
            redis: { port: PORT, host: HOST },
        });
        pushQueue.process(pushProcess);
    } catch (error) {
        const errorMessage = "Failed to setup push queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0205", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export const sendPush = (subscription: PushSubscription, payload: PushPayload, delay = 0) => {
    pushQueue.add({
        ...payload,
        ...subscription,
    }, { delay });
};
