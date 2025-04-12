import { Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { ExportConfig } from "../../builders/importExport.js";
import { BaseQueue } from "../base/queue.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper.js";

export type ExportProcessPayload = {
    config: ExportConfig;
};

let logger: winston.Logger;
let exportProcess: (job: Bull.Job<ExportProcessPayload>) => Promise<unknown>;
export let exportQueue: BaseQueue<ExportProcessPayload>;
const FOLDER = "export";

// Call this on server startup
export async function setupExportQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).getRedisUrl();
        exportProcess = (await import(getProcessPath(FOLDER))).exportProcess;

        // Initialize the Bull queue
        exportQueue = new BaseQueue<ExportProcessPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
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
}

export function startDataExport(data: ExportProcessPayload): Promise<Success> {
    return addJobToQueue(exportQueue.getQueue(), data, {});
}
