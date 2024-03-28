import Bull from "bull";
import winston from "winston";

export type ExportProcessPayload = {

}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let exportProcess: (job: Bull.Job<ExportProcessPayload>) => Promise<unknown>;
let exportQueue: Bull.Queue<ExportProcessPayload>;

// Call this on server startup
export async function setupExportQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        exportProcess = processModule.exportProcess;

        // Initialize the Bull queue
        exportQueue = new Bull<ExportProcessPayload>("export", {
            redis: { port: PORT, host: HOST },
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

export const exportData = (data: ExportProcessPayload) => {
    exportQueue.add(data); //TODO
};
