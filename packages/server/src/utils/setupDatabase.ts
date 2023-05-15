
// Sets up database on server initialization
import { init } from "../db/seeds/init.js";
import { withPrisma } from "./withPrisma.js";

export const setupDatabase = async () => {
    // Seed database
    const success = await withPrisma({
        process: async (prisma) => {
            await init(prisma);
        },
        trace: "0011",
    });
    if (!success) process.exit(1);
};
