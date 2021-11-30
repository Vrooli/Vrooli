import pkg from '@prisma/client';
import { Request } from 'express';

// Request type
declare global {
    namespace Express {
        interface Request {
            validToken: boolean | undefined;
            customerId: string | null | undefined;
            roles: string[] | undefined;
            isCustomer: boolean | undefined;
        }
    }
}

// Prisma type shorthand
export type PrismaType = pkg.PrismaClient<pkg.Prisma.PrismaClientOptions, never, pkg.Prisma.RejectOnNotFound | pkg.Prisma.RejectPerOperation | undefined>