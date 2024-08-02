import { Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper";

export type ExportProcessPayload = {
    /** What data should be exported */
    flags: {
        all: boolean;
        account: boolean;
        apis: boolean;
        bookmarks: boolean;
        bots: boolean;
        chats: boolean;
        codes: boolean;
        comments: boolean;
        issues: boolean;
        notes: boolean;
        //posts: boolean;
        pullRequests: boolean;
        projects: boolean;
        questions: boolean;
        questionAnswers: boolean;
        // quizzes: boolean;
        // quizResponses: boolean; // Includes attempts, answers, etc.
        reactions: boolean;
        reminders: boolean;
        reports: boolean;
        routines: boolean;
        runs: boolean;
        schedules: boolean;
        standards: boolean;
        // tags: boolean;
        /** Includes meetings and roles */
        teams: boolean;
        views: boolean;
    };
    /** What should happen to the exported data */
    requestType: "Delete" | "Download" | "DownloadAndDelete";
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
};

let logger: winston.Logger;
let exportProcess: (job: Bull.Job<ExportProcessPayload>) => Promise<unknown>;
let exportQueue: Bull.Queue<ExportProcessPayload>;
const FOLDER = "export";

// Call this on server startup
export async function setupExportQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).REDIS_URL;
        exportProcess = (await import(getProcessPath(FOLDER))).exportProcess;

        // Initialize the Bull queue
        exportQueue = new Bull<ExportProcessPayload>(FOLDER, {
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

export function exportData(data: ExportProcessPayload): Promise<Success> {
    return addJobToQueue(exportQueue, data, {});
}
