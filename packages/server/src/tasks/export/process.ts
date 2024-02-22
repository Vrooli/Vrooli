import { Job } from "bull";
import { logger } from "../../events/logger";
import { ExportProcessPayload } from "./queue";

export const exportProcess = async (job: Job<ExportProcessPayload>) => {
    try {
        //TODO
    } catch (err) {
        logger.error("Error exporting data", { trace: "0499" });
    }
};
