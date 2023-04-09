import { GraphQLResolveInfo } from "graphql";
import { GqlModelType, PageInfo, SessionUser, TimeFrame, VisibilityType } from '@shared/consts';
import { SingleOrArray } from "../types";

/**
 * Recursively pads object with "select" fields
 */
export type WithSelect<T> = { select: { [K in keyof T]: T[K] extends object ? WithSelect<T[K]> : true } };

/**
 * Shape 1 of 4 for GraphQL to Prisma conversion (i.e. GraphQL data before conversion)
 */
export type GraphQLInfo = GraphQLResolveInfo | { [x: string]: any } | null;

/**
 * Shape 2 of 4 for GraphQL to Prisma converstion. Used by many functions because it is more 
 * convenient than straight up GraphQL request data. Each level contains a __typename field. 
 * This type of data is also easier to hard-code in a pinch.
 */
export interface PartialGraphQLInfo {
    [x: string]: `${GqlModelType}` | undefined | boolean | PartialGraphQLInfo;
    __typename?: `${GqlModelType}`;
}

/**
 * Shape 3 of 4 for GraphQL to Prisma conversion. Still contains the type fields, 
 * but does not pad objects with a "select" field. Calculated fields, join tables, and other 
 * data transformations from the GraphqL shape are removed. This is useful when checking 
 * which fields are requested from a Prisma query.
 */
export type PartialPrismaSelect = { __typename?: `${GqlModelType}` } & { [x: string]: boolean | PartialPrismaSelect };

/**
 * Shape 4 of 4 for GraphQL to Prisma conversion. This is the final shape of the requested data 
 * as it will be sent to the database. It is has type fields removed, and objects padded with "select"
 */
export type PrismaSelect = {
    select: { [key: string]: boolean | PrismaSelectInside }
}

type PrismaSelectInside = Omit<PrismaSearch, 'select'> & {
    select: { [x: string]: boolean | PrismaSelectInside }
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
    create?: SingleOrArray<{
        [x: string]: boolean | string | number | PrismaCreateInside
    }>;
    connect?: SingleOrArray<{
        [x: string]: boolean | string | number | PrismaCreateInside
    }>;
}

export type PrismaCreate = {
    [x: string]: boolean | string | number | PrismaCreateInside
}

type PrismaUpdateInside = {
    create?: SingleOrArray<{
        [x: string]: boolean | string | number | PrismaUpdateInside
    }>;
    connect?: {
        where: any;
    };
    update?: SingleOrArray<{
        [x: string]: boolean | string | number | PrismaUpdateInside
    }>;
    delete?: SingleOrArray<{
        where: any;
    }>;
    disconnect?: SingleOrArray<{
        where: any;
    }>;
}

export type PrismaUpdate = {
    id: string;
    [x: string]: boolean | string | number | PrismaUpdateInside
}

export type RelationshipType = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Update';

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

// Optional if IsRequired is false
type MaybeOptional<T extends { [x: string]: any }, IsRequired extends boolean> =
    IsRequired extends true ? T : T | undefined;

export type BuiltRelationship<
    IDField extends string,
    IsAdd extends boolean,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    Shaped extends { [x: string]: any },
> = MaybeOptional<(
    IsAdd extends true ?
    IsOneToOne extends true ?
    {
        connect?: RelConnect<IDField>,
        create?: RelCreate<Shaped>,
    } : {
        connect?: RelConnect<IDField>[],
        create?: RelCreate<Shaped>[],
    }
    : IsOneToOne extends true ?
    {
        connect?: RelConnect<IDField>,
        disconnect?: boolean,
        delete?: boolean,
        create?: RelCreate<Shaped>,
        update?: RelUpdate<Shaped, IDField>['data'],
    } : {
        connect?: RelConnect<IDField>[],
        disconnect?: RelDisconnect<IDField>[],
        delete?: RelDelete<IDField>[],
        create?: RelCreate<Shaped>[],
        update?: RelUpdate<Shaped, IDField>[],
    }
), IsRequired>

export type VisibilityBuilderProps = {
    objectType: `${GqlModelType}`,
    userData: SessionUser | null | undefined,
    visibility?: VisibilityType | null | undefined,
}