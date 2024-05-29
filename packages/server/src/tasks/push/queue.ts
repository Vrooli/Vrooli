import { HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { addJobToQueue } from "../queueHelper";

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
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupPushQueue = async () => {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        pushProcess = processModule.pushProcess;

        // Initialize the Bull queue
        pushQueue = new Bull<PushSubscription & PushPayload>("push", {
            redis: { port: PORT, host: HOST },
            defaultJobOptions: {
                removeOnComplete: {
                    age: HOURS_1_S,
                    count: 10_000,
                },
                removeOnFail: {
                    age: HOURS_1_S,
                    count: 10_000,
                },
            },
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
};

export const sendPush = (
    subscription: PushSubscription,
    payload: PushPayload,
    delay = 0,
): Promise<Success> => {
    return addJobToQueue(pushQueue, {
        ...payload,
        ...subscription,
    }, { delay });
};
