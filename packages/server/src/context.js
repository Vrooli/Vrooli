import pkg from '@prisma/client';
const { PrismaClient } = pkg;

const prisma = new PrismaClient()

export const context = ({ req, res }) => ({
    prisma,
    req,
    res
})