import { HOURS_1_S } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";

export type ImportProcessPayload = {

}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let importProcess: (job: Bull.Job<ImportProcessPayload>) => Promise<unknown>;
let importQueue: Bull.Queue<ImportProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupImportQueue = async () => {
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
        importProcess = processModule.importProcess;

        // Initialize the Bull queue
        importQueue = new Bull<ImportProcessPayload>("import", {
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
        importQueue.process(importProcess);
    } catch (error) {
        const errorMessage = "Failed to setup import queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0209", error });
        } else {
            console.error(errorMessage, error);
        }
    }
};

export const importData = (data: ImportProcessPayload) => {
    importQueue.add(data); //TODO
};
