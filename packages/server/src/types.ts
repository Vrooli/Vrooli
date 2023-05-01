import { FocusModeStopCondition, GqlModelType, SessionUser } from "@local/shared";
import pkg from "@prisma/client";
import { GraphQLResolveInfo } from "graphql";
import { Context } from "./middleware";

export interface BasicToken {
    iat: number;
    iss: string;
    exp: number;
}
export interface ApiToken extends BasicToken {
    apiToken: string;
}

// Tokens store more limited data than the session object returned by the API. 
// This is because the maximum size of a cookie is 4kb
export interface SessionToken extends BasicToken {
    isLoggedIn: boolean;
    timeZone?: string;
    // Supports logging in with multiple accounts
    users: SessionUserToken[];
}
export type SessionUserToken = Pick<SessionUser, "id" | "handle" | "hasPremium" | "languages" | "name"> & {
    activeFocusMode?: {
        mode: { id: string; },
        stopCondition: FocusModeStopCondition,
        stopTime?: Date | null,
    } | null;
}

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
            users?: SessionUserToken[];
            validToken?: boolean;
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
 * Return type of find one queries
 */
export type FindOneResult<T> = RecursivePartial<T> | null

/**
 * Return type of find many queries
 */
export type FindManyResult<T> = {
    pageInfo: {
        hasNextPage: boolean,
        endCursor?: string | null
    },
    edges: Array<{ cursor: string, node: RecursivePartial<T> }>
}

/**
 * Return type of create one mutations
 */
export type CreateOneResult<T> = FindOneResult<T>

/**
 * Return type of update one mutations
 */
export type UpdateOneResult<T> = FindOneResult<T>

export type GQLEndpoint<T, U> = (parent: undefined, data: IWrap<T>, context: Context, info: GraphQLResolveInfo) => Promise<U>;

export type UnionResolver = { __resolveType: (obj: any) => `${GqlModelType}` };

export type SingleOrArray<T> = T | T[];

/**
 * Either a promise or a value
 */
export type PromiseOrValue<T> = Promise<T> | T;

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
