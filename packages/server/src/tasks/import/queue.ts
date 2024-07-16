import { HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { addJobToQueue } from "../queueHelper";

export type ImportProcessPayload = {

}

let logger: winston.Logger;
let importProcess: (job: Bull.Job<ImportProcessPayload>) => Promise<unknown>;
let importQueue: Bull.Queue<ImportProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export async function setupImportQueue() {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        const REDIS_URL = redisConnModule.REDIS_URL;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        importProcess = processModule.importProcess;

        // Initialize the Bull queue
        importQueue = new Bull<ImportProcessPayload>("import", {
            redis: REDIS_URL,
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
        importQueue.process(importProcess);
    } catch (error) {
        const errorMessage = "Failed to setup import queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0209", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export function importData(data: ImportProcessPayload): Promise<Success> {
    return addJobToQueue(importQueue, data, {}); //TODO
}
