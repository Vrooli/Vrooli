
// Sets up database on server initialization
import pkg from '@prisma/client';
import { init } from '../db/seeds/init.js';
import { genErrorCode, logger, LogLevel } from '../logger';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;
const prisma = new PrismaClient();

const executeSeed = async (func: (prisma: PrismaType) => any, exitOnFail = false,) => {
    await func(prisma).catch((error: any) => {
        logger.log(LogLevel.error, 'Database seed caught error', { code: genErrorCode('00011'), error });
        if (exitOnFail) process.exit(1);
    }).finally(async () => {
        await prisma.$disconnect();
    })
}

export const setupDatabase = async () => {
    // Seed database
    await executeSeed(init, true);
}