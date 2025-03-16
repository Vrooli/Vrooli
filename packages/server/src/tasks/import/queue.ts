import { MINUTES_5_MS, Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { ImportConfig, ImportData } from "../../builders/importExport.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper.js";

export type ImportProcessPayload = {
    data: ImportData;
    config: ImportConfig;
}

let logger: winston.Logger;
let importProcess: (job: Bull.Job<ImportProcessPayload>) => Promise<unknown>;
let importQueue: Bull.Queue<ImportProcessPayload>;
const FOLDER = "import";

/**
 * How long the Bull queue will wait before giving and marking it as failed.
 */
const JOB_TIMEOUT_MS = MINUTES_5_MS;

// Call this on server startup
export async function setupImportQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).getRedisUrl();
        importProcess = (await import(getProcessPath(FOLDER))).importProcess;

        // Initialize the Bull queue
        importQueue = new Bull<ImportProcessPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
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

export function startDataImport(data: ImportProcessPayload): Promise<Success> {
    return addJobToQueue(importQueue, data, { timeout: JOB_TIMEOUT_MS });
}
