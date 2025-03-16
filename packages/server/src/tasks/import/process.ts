import { Job } from "bull";
import { importData } from "../../builders/importExport.js";
import { logger } from "../../events/logger.js";
import { ImportProcessPayload } from "./queue.js";

export async function importProcess(job: Job<ImportProcessPayload>) {
    try {
        const { data, config } = job.data;
        const result = await importData(data, config);
        return result;
    } catch (err) {
        logger.error("Error importing data", { trace: "0500" });
    }
}
