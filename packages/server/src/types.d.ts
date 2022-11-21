import pkg from '@prisma/client';
import { SessionUser } from './schema/types';
import awardLocale from './locales/en/award.json';
import commonLocale from './locales/en/common.json';
import errorLocale from './locales/en/error.json';
import notifyLocale from './locales/en/notify.json';
import validateLocale from './locales/en/validate.json';

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
             * Preferred languages to display errors, push notifications, etc. in. 
             * Always has at least one language
             */
            languages: string[];
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

// Internationlization
declare module "i18next" {
    interface CustomTypeOptions {
        defaultNS: 'common';
        resources: {
            award: typeof awardLocale;
            common: typeof commonLocale;
            validate: typeof validateLocale;
            error: typeof errorLocale;
            notify: typeof notifyLocale;
        };
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

export type SingleOrArray<T> = T | T[];

/**
 * Type for replacing one type with another in a nested object
 */
export type ReplaceTypes<ObjType extends object, FromType, ToType> = {
    [KeyType in keyof ObjType]: ObjType[KeyType] extends object
    ? ReplaceTypes<ObjType[KeyType], FromType, ToType>
    : ObjType[KeyType] extends FromType
    ? ToType
    : ObjType[KeyType];
}