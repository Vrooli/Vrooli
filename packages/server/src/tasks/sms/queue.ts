import { BUSINESS_NAME, HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { CustomError } from "../../events/error";
import { addJobToQueue } from "../queueHelper";

export type SmsProcessPayload = {
    to: string[];
    body: string;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let smsProcess: (job: Bull.Job<SmsProcessPayload>) => Promise<unknown>;
let smsQueue: Bull.Queue<SmsProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupSmsQueue = async () => {
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
        smsProcess = processModule.smsProcess;

        // Initialize the Bull queue
        smsQueue = new Bull<SmsProcessPayload>("sms", {
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
        smsQueue.process(smsProcess);
    } catch (error) {
        const errorMessage = "Failed to setup sms queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0211", error });
        } else {
            console.error(errorMessage, error);
        }
    }
};

export const sendSms = (to: string[], body: string): Promise<Success> => {
    // Must include at least one "to" number
    if (to.length === 0) {
        throw new CustomError("0353", "InternalError", ["en"]);
    }
    return addJobToQueue(smsQueue, { to, body }, {});
};

/** Adds a verification code text to a task queue */
export const sendSmsVerification = (phoneNumber: string, code: string): Promise<Success> => {
    return addJobToQueue(smsQueue, {
        to: [phoneNumber],
        body: `${code} is your ${BUSINESS_NAME} verification code`,
    }, {});
};
