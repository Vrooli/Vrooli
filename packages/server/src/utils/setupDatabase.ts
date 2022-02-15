
// Sets up database on server initialization. For this project, this includes:  
// 1. Seeding database with required data
// 2. Seeding database with mock data
import pkg from '@prisma/client';
import { init } from '../db/seeds/init.js';
import { mock } from '../db/seeds/mock.js';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;
const prisma = new PrismaClient();

const executeSeed = async (func: (prisma: PrismaType) => any, exitOnFail = false,) => {
    await func(prisma).catch((error: any) => {
        console.error(error);
        if (exitOnFail) process.exit(1);
    }).finally(async () => {
        await prisma.$disconnect();
    })
}

export const setupDatabase = async () => {
    // Seed database
    await executeSeed(init, true);
    if (process.env.CREATE_MOCK_DATA) {
        await executeSeed(mock, false);
    }
}