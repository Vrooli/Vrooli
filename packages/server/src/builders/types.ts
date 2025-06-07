import { type ModelType, type OrArray, type PageInfo, type TimeFrame, type VisibilityType } from "@vrooli/shared";
import { type RequestService } from "../auth/request.js";

/**
 * Shape 2 of 4 for API endpoint to Prisma conversion. Each level contains a __typename field. 
 * This type of data is also easier to hard-code in a pinch.
 */
export interface PartialApiInfo {
    [x: string]: string | `${ModelType}` | undefined | boolean | PartialApiInfo; // string and `${ModelType}` only included for __typename (known TypeScript limitation: https://github.com/microsoft/TypeScript/issues/27144)
    __typename?: `${ModelType}`;
    __cacheKey?: string;
}

/**
 * Shape 4 of 4 for API endpoint to Prisma conversion. This is the final shape of the requested data 
 * as it will be sent to the database. It is has type fields removed, and objects padded with "select"
 */
export type PrismaSelect = {
    select: { [key: string]: boolean | PrismaSelectInside }
}

type PrismaSelectInside = Omit<PrismaSearch, "select" | "where"> & {
    select: { [x: string]: boolean | PrismaSelectInside };
    where?: Record<string, unknown>;
}

export type PrismaSearch = {
    where: any;
    select?: any;
    orderBy?: any;
    cursor?: any;
    take?: any;
    skip?: any;
    distinct?: any;
}

type PrismaCreateInside = {
    create?: OrArray<{
        [x: string]: boolean | string | number | PrismaCreateInside
    }>;
    connect?: OrArray<{
        [x: string]: boolean | string | number | PrismaCreateInside
    }>;
}

export type PrismaCreate = {
    [x: string]: boolean | string | number | PrismaCreateInside
}

type PrismaUpdateInside = {
    create?: OrArray<{
        [x: string]: boolean | string | number | PrismaUpdateInside
    }>;
    connect?: {
        where: any;
    };
    update?: OrArray<{
        [x: string]: boolean | string | number | PrismaUpdateInside
    }>;
    delete?: OrArray<{
        where: any;
    }>;
    disconnect?: OrArray<{
        where: any;
    }>;
}

export type PrismaUpdate = {
    id: string;
    [x: string]: boolean | string | number | PrismaUpdateInside
}

export type RelationshipType = "Connect" | "Create" | "Delete" | "Disconnect" | "Update";

/**
 * Generic Prisma model type. Useful for helper functions that work with any model
 */
export interface PrismaDelegate {
    findUnique: (args: { where: any, select?: any }) => Promise<{ [x: string]: any } | null>;
    findFirst: (args: PrismaSearch) => Promise<{ [x: string]: any } | null>;
    findMany: (args: PrismaSearch) => Promise<{ [x: string]: any }[]>;
    create: (args: { data: any, select?: any }) => Promise<{ [x: string]: any }>;
    update: (args: { data: any, where: any, select?: any }) => Promise<{ [x: string]: any }>;
    upsert: (args: {
        create: any;
        update: any;
        where: any;
        select?: any;
    }) => Promise<{ [x: string]: any }>;
    delete: (args: { where: any }) => Promise<{ [x: string]: any }>;
    deleteMany: (args: { where: any }) => Promise<{ count: number }>;
    count: (args: {
        where: any;
        cursor?: any;
        take?: any;
        skip?: any;
        orderBy?: any;
        select?: any;
    }) => Promise<number>;
    aggregate: (args: {
        where: any;
        orderBy?: any;
        cursor?: any;
        skip?: any;
        take?: any;
        _count?: any;
        _avg?: any;
        _sum?: any;
        _min?: any;
        _max?: any;
    }) => Promise<any>;
}

export type SelectWrap<T extends Record<string, any>> = { select: { [K in keyof Required<T>]: true } }

export type PaginatedSearchResult = {
    __typename: any;
    pageInfo: PageInfo;
    edges: Array<{
        __typename: any;
        cursor: string;
        node: any;
    }>;
}

export type SearchInputBase<SortBy> = {
    ids?: string[] | null; // Specific ids to search for
    searchString?: string | null; // String to search for. Which fields this includes are defined by the model
    sortBy?: SortBy | null; // Sort order
    createdTimeFrame?: Partial<TimeFrame> | null; // Objects created within this timeFrame
    updatedTimeFrame?: Partial<TimeFrame> | null; // Objects updated within this timeFrame
    after?: string | null;
    take?: number | null;
}


export type CountInputBase = {
    createdTimeFrame?: Partial<TimeFrame> | null;
    updatedTimeFrame?: Partial<TimeFrame> | null;
}

export type ExceptionsBuilderProps = {
    /**
     * Fields that are allowed to be queried. Supports nested fields through dot notation
     */
    canQuery: string[],
    /**
     * Default for main field
     */
    defaultValue?: any,
    /**
     * Field to check for stringified exceptions
     */
    exceptionField: string,
    /**
     * Input object, with exceptions in one of the fields
     */
    input: { [x: string]: any },
    /**
     * Main field being queried
     */
    mainField: string,
}

export type ExistsArrayProps = {
    ids: (string | null | undefined)[],
    prismaDelegate: any,
    where: { [x: string]: any },
}

export type RelConnect<IDField extends string> = { [key in IDField]: string }
export type RelDisconnect<IDField extends string> = { [key in IDField]: string }
export type RelCreate<Shaped extends { [x: string]: any }> = Shaped
export type RelUpdate<Shaped extends { [x: string]: any }, IDField extends string> = { where: { [key in IDField]: string }, data: Shaped }
export type RelDelete<IDField extends string> = { [key in IDField]: string }

export type VisibilityBuilderProps = {
    objectType: `${ModelType}`,
    req: Parameters<typeof RequestService.assertRequestFrom>[0],
    searchInput: { [x: string]: any };
    visibility?: VisibilityType | null | undefined,
}

export type VisibilityBuilderPrismaResult = {
    query: { [x: string]: any };
    visibilityUsed: VisibilityType;
}
