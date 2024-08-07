import { SessionUserToken } from "@local/server";
import { HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { addJobToQueue } from "../queueHelper";

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
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export async function setupExportQueue() {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        const REDIS_URL = redisConnModule.REDIS_URL;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        exportProcess = processModule.exportProcess;

        // Initialize the Bull queue
        exportQueue = new Bull<ExportProcessPayload>("export", {
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
