import pkg from '@prisma/client';
import { SessionUser } from './schema/types';

// Request type
declare global {
    namespace Express {
        interface Request {
            /**
             * Public API token, if present
             */
             apiToken?: boolean;
            /**
             * True if the request is coming from a safe origin (e.g. our own frontend)
             */
            fromSafeOrigin?: boolean;
            /**
             * True if user is logged in. False if not, or if token is invalid or for an API token
             */
            isLoggedIn?: boolean;
            /**
             * User's current time zone
             */
            timeZone?: string;
            /**
             * Users logged in with this session (if isLoggedIn is true)
             */
            users?: SessionUser[];
            validToken?: boolean;
        }
    }
}

export type ReqForUserAuth = { users?: { id?: string | null }[] }

/**
 * Prisma type shorthand
 */
export type PrismaType = pkg.PrismaClient<pkg.Prisma.PrismaClientOptions, never, pkg.Prisma.RejectOnNotFound | pkg.Prisma.RejectPerOperation | undefined>

/**
 * Wrapper for GraphQL input types
 */
export type IWrap<T> = { input: T }

/**
 * Type for converting GraphQL objects (where nullables are set based on database), 
 * to fully OPTIONAL objects (including relationships)
 */
export type RecursivePartial<T> = {
    [P in keyof T]?: T[P] extends Date
    ? T[P]
    : T[P] extends object
    ? RecursivePartial<T[P]>
    : T[P]
};

/**
 * Generic Prisma model type. Useful for helper functions that work with any model
 */
export interface PrismaDelegate {
    findUnique: (args: any) => Promise<any>;
    findFirst: (args: any) => Promise<any>;
    findMany: (args: any) => Promise<any>;
    create: (args: any) => Promise<any>;
    createMany: (args: any) => Promise<any>;
    update: (args: any) => Promise<any>;
    updateMany: (args: any) => Promise<any>;
    upsert: (args: any) => Promise<any>;
    delete: (args: any) => Promise<any>;
    deleteMany: (args: any) => Promise<any>;
    count: (args: any) => Promise<any>;
    aggregate: (args: any) => Promise<any>;
}