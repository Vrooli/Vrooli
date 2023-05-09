import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";
import { generateEmbeddings } from "./generateEmbeddings";

/**
 * Initializes cron jobs for generating text embeddings
 */
export const initGenerateEmbeddingsCronJob = () => {
    initializeCronJob(cronTimes.embeddings, generateEmbeddings, "generate embeddings");
};
