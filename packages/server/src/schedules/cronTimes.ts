/**
 * Stores the cron times for each cron job. 
 * Cron syntax created using this website: https://crontab.guru/
 * 
 * NOTE: Cron jobs start at a weird time so they run when there is less activity, 
 * and when other cron jobs are not running.
 */
export const cronTimes = {
    counts: "49 5 1 * *", // Every month at 5:49am (UTC)
    embeddings: "6 * * * *", // Every hour at the 6th minute
    events: "9 5 * * *", // Every day at 5:09am (UTC)
    expirePremium: "20 4 * * *", // Every day at 4:20am (UTC)
    pullRequests: "58 4 * * *", // Every day at 4:58am (UTC)
    reports: "56 4 * * *", // Every day at 4:56am (UTC)
    sitemaps: "43 4 * * *", // Every day at 4:43am (UTC)
};
