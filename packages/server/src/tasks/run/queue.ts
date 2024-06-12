import { HOURS_1_S, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { addJobToQueue } from "../queueHelper";

export type RunProjectPayload = {
    __process: "Project",
    //...
};

export type RunRoutinePayload = {
    __process: "Routine",
    //...
};

export type RunPayload = RunProjectPayload | RunRoutinePayload;

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let runProcess: (job: Bull.Job<RunPayload>) => Promise<unknown>;
let runQueue: Bull.Queue<RunPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupRunQueue = async () => {
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
        runProcess = processModule.runProcess;

        // Initialize the Bull queue
        runQueue = new Bull<RunPayload>("run", {
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
        runQueue.process(runProcess);
    } catch (error) {
        const errorMessage = "Failed to setup run queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0567", error });
        } else {
            console.error(errorMessage, error);
        }
    }
};

export const processRunProject = (
    props: Omit<RunProjectPayload, "__process">,
): Promise<Success> => {
    return addJobToQueue(runQueue, props, {});
};

export const processRunRoutine = (
    props: Omit<RunRoutinePayload, "__process">,
): Promise<Success> => {
    return addJobToQueue(runQueue, props, {});
};
