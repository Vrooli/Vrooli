import { ModelType, SessionUser } from "@local/shared";
import pkg from "@prisma/client";
import { PartialApiInfo } from ".";
import { Context } from "./middleware";

declare module "@local/server";
export * from ".";

/**
 * Information required in any JWT token
 */
export interface BasicToken {
    iat: number;
    iss: string;
    exp: number;
}
export interface AccessToken extends BasicToken {
    /** When we need to check the database to see if the token is still valid. */
    accessExpiresAt: number;
}
export interface ApiToken extends AccessToken {
    apiToken: string;
}

/**
 * Information stored in a session token.
 * Contains less data than the full session object, since it is stored in a cookie 
 * and the maximum size of a cookie is 4kb.
 */
export interface SessionToken extends AccessToken {
    isLoggedIn: boolean;
    timeZone?: string;
    // Supports logging in with multiple accounts
    users: SessionUser[];
}

/** All data stored in session */
export type SessionData = {
    /** When we need to check the database to see if the token is still valid. */
    accessExpiresAt?: number | null;
    /** Public API token, if present */
    apiToken?: boolean;
    /** True if the request is coming from a safe origin (e.g. our own frontend) */
    fromSafeOrigin?: boolean;
    /** True if user is logged in. False if not, or if token is invalid or for an API token */
    isLoggedIn?: boolean;
    /**
     * Preferred languages to display errors, push notifications, etc. in. 
     * Always has at least one language
     */
    languages?: string[];
    /** User's current time zone */
    timeZone?: string | null;
    /** Users logged in with this session (if isLoggedIn is true) */
    users?: SessionUser[] | null;
    validToken?: boolean;
}

// Add session to socket
declare module "socket.io" {
    export interface Socket {
        req: {
            ip: string;
        };
        session: SessionData;
    }
}

declare module "winston" {
    // Add "warning" to the list of levels
    interface Logger {
        warning: winston.LeveledLogMethod;
    }
}

export type RequestFile = {
    fieldname: string;
    originalname: string;
    encoding: string;
    mimetype: string;
    buffer: Buffer;
    size: number;
}

// Request type
declare global {
    namespace Express {
        interface Request {
            files?: RequestFile[];
            session: SessionData;
        }
    }
}

export type WithIdField<IdField extends string = "id"> = {
    [key in IdField]: string;
}

/** Prisma type shorthand */
export type PrismaType = pkg.PrismaClient<pkg.Prisma.PrismaClientOptions, never, pkg.Prisma.RejectOnNotFound | pkg.Prisma.RejectPerOperation | undefined>

/** Wrapper for API endpoint input types */
export type IWrap<T> = { input: T }

/**
 * Type for converting API endpoint objects (where nullables are set based on database), 
 * to fully OPTIONAL objects (including relationships)
 */
export type RecursivePartial<T> = {
    [P in keyof T]?: T[P] extends Date
    ? T[P]
    : T[P] extends object
    ? RecursivePartial<T[P]>
    : T[P]
};

/** Like RecursivePartial, but allows `null` */
export type RecursivePartialNullable<T> = {
    [P in keyof T]?: T[P] extends Date
    ? T[P]
    : T[P] extends object
    ? RecursivePartialNullable<T[P]>
    : T[P] | null
};

export type ApiEndpoint<T, U> = (
    data: T extends undefined ? undefined : IWrap<T>,
    context: Context,
    info: PartialApiInfo
) => Promise<U extends Array<infer V> ? RecursivePartial<V>[] : RecursivePartial<U>>;

export type UnionResolver = { __resolveType: (obj: any) => `${ModelType}` };

/** Either a promise or a value */
export type PromiseOrValue<T> = Promise<T> | T;

/** Type for replacing one type with another in a nested object */
export type ReplaceTypes<ObjType extends object, FromType, ToType> = {
    [KeyType in keyof ObjType]: ObjType[KeyType] extends object
    ? ReplaceTypes<ObjType[KeyType], FromType, ToType>
    : ObjType[KeyType] extends FromType
    ? ToType
    : ObjType[KeyType];
}
