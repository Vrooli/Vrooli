/**
 * Creates text embeddings for all searchable objects, which either:
 * - Don't have embeddings yet
 * - Have been updated since their last embedding was created
 */
import { cronTimes } from "../cronTimes";
import { initializeCronJob } from "../initializeCronJob";
import { generateEmbeddings } from "./generateEmbeddings";

/**
 * Initializes cron jobs for generating text embeddings
 */
export const initGenerateEmbeddingsCronJob = () => {
    initializeCronJob(cronTimes.embeddings, generateEmbeddings, "generate embeddings");
};
