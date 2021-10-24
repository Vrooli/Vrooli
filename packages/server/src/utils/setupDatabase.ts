
// Sets up database on server initialization. For this project, this includes:  
// 1. Cleaning any user-uploaded images that no longer exist in the database  
// 2. Seeding database with required data
// 3. Seeding database with mock data
import { cleanImageData } from "./cleanImageData.js";
import pkg from '@prisma/client';
import { init } from '../db/seeds/init.js';
import { mock } from '../db/seeds/mock.js';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;
const prisma = new PrismaClient();

const executeSeed = async (func: (prisma: PrismaType) => any) => {
    await func(prisma).catch((error: any) => {
        console.error(error);
        process.exit(1);
    }).finally(async () => {
        await prisma.$disconnect();
    })
}

export const setupDatabase = async () => {
    // 1. Clean old images
    await cleanImageData();
    // 2. Seed database
    await executeSeed(init);
    if (process.env.CREATE_MOCK_DATA) {
        await executeSeed(mock);
    }
}