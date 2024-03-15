import { BUSINESS_NAME } from "@local/shared";
import Bull from "bull";
import winston from "winston";

export type SmsProcessPayload = {
    to: string[];
    body: string;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let smsProcess: (job: Bull.Job<SmsProcessPayload>) => Promise<unknown>;
let smsQueue: Bull.Queue<SmsProcessPayload>;

// Call this on server startup
export async function setupSmsQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        smsProcess = processModule.smsProcess;

        // Initialize the Bull queue
        smsQueue = new Bull<SmsProcessPayload>("sms", {
            redis: { port: PORT, host: HOST },
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
}

export function sendSms(to = [], body: string) {
    smsQueue.add({ to, body });
}

/** Adds a verification code text to a task queue */
export const sendSmsVerification = (phoneNumber: string, code: string) => {
    smsQueue.add({
        to: [phoneNumber],
        body: `${code} is your ${BUSINESS_NAME} verification code`,
    });
};
