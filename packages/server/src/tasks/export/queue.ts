import { HOURS_1_S } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";

export type ExportProcessPayload = {

}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let exportProcess: (job: Bull.Job<ExportProcessPayload>) => Promise<unknown>;
let exportQueue: Bull.Queue<ExportProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupExportQueue = async () => {
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
        exportProcess = processModule.emailProcess;

        // Initialize the Bull queue
        exportQueue = new Bull<ExportProcessPayload>("export", {
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
        exportQueue.process(exportProcess);
    } catch (error) {
        const errorMessage = "Failed to setup export queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0206", error });
        } else {
            console.error(errorMessage, error);
        }
    }
};

export const exportData = (data: ExportProcessPayload) => {
    exportQueue.add(data); //TODO
};
