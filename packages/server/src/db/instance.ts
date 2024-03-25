import pkg from "@prisma/client";

const { PrismaClient } = pkg;
export const prismaInstance = new PrismaClient();
