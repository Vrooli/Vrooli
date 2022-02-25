import pkg from '@prisma/client';
import { Request } from 'express';
import { Maybe } from 'schema/types';

// Request type
declare global {
    namespace Express {
        interface Request {
            validToken?: boolean;
            userId?: string;
            roles?: string[];
            isLoggedIn?: boolean;
        }
    }
}

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
 * Type that defines a state between a GraphQL to Prisma type conversion, where: 
 * - Calculated/virtual fields are not yet handled
 * - Unions are not yet handled
 * - Join tables are not yet handled
 * The function using this type should be able to make the conversion
 * NOTE: "select" padding is handled later, so no need to handle it here
 * @param T GraphQL object type being converted
 */
export type PartialSelectConvert<T> = any;
// export type PartialSelectConvert<T> = T extends Date | string | number | boolean | null
//     ? boolean
//     : {
//         [P in keyof T]?: T[P] extends Date | string | number | boolean | null
//         ? boolean
//         : T[P] extends Array<T>
//         ? { [x: string]: boolean } // Prevent infinite recursion
//         : T[P] extends Array<any>
//         ? PartialSelectConvert<T[P][number]>
//         : T[P] extends T
//         ? { [x: string]: boolean } // Prevent infinite recursion
//         : PartialSelectConvert<T[P]>
//     };