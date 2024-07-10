import { HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { addJobToQueue } from "../queueHelper";
import { SandboxProcessPayload } from "./types";

let logger: winston.Logger;
let sandboxProcess: (job: Bull.Job<SandboxProcessPayload>) => Promise<unknown>;
let sandboxQueue: Bull.Queue<SandboxProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export async function setupSandboxQueue() {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        const REDIS_URL = redisConnModule.REDIS_URL;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        sandboxProcess = processModule.sandboxProcess;

        // Initialize the Bull queue
        sandboxQueue = new Bull<SandboxProcessPayload>("sandbox", {
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
        sandboxQueue.process(sandboxProcess);
    } catch (error) {
        const errorMessage = "Failed to setup sandbox queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0553", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export function runSandboxedCode(data: SandboxProcessPayload): Promise<Success> {
    return addJobToQueue(sandboxQueue, data, {});
}
