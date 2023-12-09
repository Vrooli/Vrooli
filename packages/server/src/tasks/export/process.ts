import { Job } from "bull";
import { logger } from "../../events/logger";
import { ExportProcessPayload } from "./queue";

export async function exportProcess(job: Job<ExportProcessPayload>) {
    try {
        //TODO
    } catch (err) {
        logger.error("Error exporting data", { trace: "0499" });
    }
}
