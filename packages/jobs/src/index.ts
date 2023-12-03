import { initCountsCronJobs, initEventsCronJobs, initExpirePremiumCronJob, initGenerateEmbeddingsCronJob, initModerationCronJobs, initSitemapCronJob, initStatsCronJobs } from "./schedules";

initStatsCronJobs();
initEventsCronJobs();
initCountsCronJobs();
initSitemapCronJob();
initModerationCronJobs();
initExpirePremiumCronJob();
initGenerateEmbeddingsCronJob();
