import pkg from "@prisma/client";
import { init } from "../db/seeds/init.js";
import { logger } from "../events/logger";
const { PrismaClient } = pkg;
const prisma = new PrismaClient();
const executeSeed = async (func, exitOnFail = false) => {
    await func(prisma).catch((error) => {
        logger.error("Database seed caught error", { trace: "0011", error });
        if (exitOnFail)
            process.exit(1);
    }).finally(async () => {
        await prisma.$disconnect();
    });
};
export const setupDatabase = async () => {
    await executeSeed(init, true);
};
//# sourceMappingURL=setupDatabase.js.map