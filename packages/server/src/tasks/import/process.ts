import { type Job } from "bullmq";
import { importData } from "../../builders/importExport.js";
import { logger } from "../../events/logger.js";
import { type ImportUserDataTask } from "../taskTypes.js";

export async function importProcess(job: Job<ImportUserDataTask>) {
    try {
        const { data, config } = job.data;
        const result = await importData(data, config);
        return result;
    } catch (err) {
        logger.error("Error importing data", { trace: "0500" });
    }
}
