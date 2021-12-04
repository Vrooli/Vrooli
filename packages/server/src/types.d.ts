import pkg from '@prisma/client';
import { Request } from 'express';

// Request type
declare global {
    namespace Express {
        interface Request {
            validToken?: boolean;
            customerId?: string;
            roles?: string[];
            isCustomer?: boolean;
        }
    }
}

// Prisma type shorthand
export type PrismaType = pkg.PrismaClient<pkg.Prisma.PrismaClientOptions, never, pkg.Prisma.RejectOnNotFound | pkg.Prisma.RejectPerOperation | undefined>