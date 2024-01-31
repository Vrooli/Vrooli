import Bull from "bull";
import winston from "winston";

export type ImportProcessPayload = {

}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let importProcess: (job: Bull.Job<ImportProcessPayload>) => Promise<unknown>;
let importQueue: Bull.Queue<ImportProcessPayload>;

// Call this on server startup
export async function setupImportQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        importProcess = processModule.importProcess;

        // Initialize the Bull queue
        importQueue = new Bull<ImportProcessPayload>("import", {
            redis: { port: PORT, host: HOST }
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

export function importData(data: ImportProcessPayload) {
    importQueue.add(data); //TODO
}
