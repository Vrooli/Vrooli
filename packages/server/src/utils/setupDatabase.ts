/** Performs database setup, including seeding */
export const setupDatabase = async () => {
    const { init } = await import("../db/seeds/init.js");
    const { withPrisma } = await import("./withPrisma.js");
    // Seed database
    const success = await withPrisma({
        process: async (prisma) => {
            await init(prisma);
        },
        trace: "0011",
    });
    if (!success) process.exit(1);
};
