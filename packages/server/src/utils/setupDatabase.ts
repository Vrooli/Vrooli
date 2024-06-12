/** Performs database setup, including seeding */
export const setupDatabase = async () => {
    const { init } = await import("../db/seeds/init.js");
    const { logger } = await import("../events/logger.js");
    // Seed database
    try {
        await init();
    } catch (error) {
        logger.error("Caught error in setupDatabase", { trace: "0011", error });
        // Don't let the app start if the database setup fails
        process.exit(1);
    }
};
