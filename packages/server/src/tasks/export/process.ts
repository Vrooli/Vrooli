import { Job } from "bull";
// import { exportData } from "../../builders/importExport.js";
import { logger } from "../../events/logger.js";
import { ExportProcessPayload } from "./queue.js";

export async function exportProcess(job: Job<ExportProcessPayload>) {
    try {
        const { config } = job.data;
        // const result = await exportData(data, config);
        // return result;
        //TODO
    } catch (err) {
        logger.error("Error exporting data", { trace: "0499" });
    }
}
