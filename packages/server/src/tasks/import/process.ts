import { Job } from "bull";
import { logger } from "../../events/logger";
import { ImportProcessPayload } from "./queue";

export async function importProcess(job: Job<ImportProcessPayload>) {
    try {
        //TODO
    } catch (err) {
        logger.error("Error importing data", { trace: "0500" });
    }
}
