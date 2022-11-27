import { GraphQLResolveInfo } from "graphql";
import { GraphQLModelType, Mutater } from "../models/types";
import { PageInfo, SessionUser, TimeFrame, VisibilityType } from "../schema/types";
import { PrismaType, SingleOrArray } from "../types";

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
    [x: string]: GraphQLModelType | undefined | boolean | PartialGraphQLInfo;
    __typename?: GraphQLModelType;
}

/**
 * Shape 3 of 4 for GraphQL to Prisma conversion. Still contains the __typename fields, 
 * but does not pad objects with a "select" field. Calculated fields, join tables, and other 
 * data transformations from the GraphqL shape are removed. This is useful when checking 
 * which fields are requested from a Prisma query.
 */
export type PartialPrismaSelect = { __typename?: GraphQLModelType } & { [x: string]: boolean | PartialPrismaSelect };

/**
 * Shape 4 of 4 for GraphQL to Prisma conversion. This is the final shape of the requested data 
 * as it will be sent to the database. It is has __typename fields removed, and objects padded with "select"
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
    [x: string]: boolean | string | number | PrismaUpdateInside
}

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

export type PaginatedSearchResult = {
    pageInfo: PageInfo;
    edges: Array<{
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

export type RelationshipTypes = 'Connect' | 'Disconnect' | 'Create' | 'Update' | 'Delete';

export interface RelationshipBuilderHelperArgs<
    IDField extends string,
    IsAdd extends boolean,
    IsOneToOne extends boolean,
    IsRequired extends boolean,
    RelName extends string,
    Input extends { [key in RelName]: any },
> {
    /**
     * The data to convert
     */
    data: Input,
    /**
     * The name of the relationship to convert. This is required because we're grabbing the 
     * relationship from its parent object
     */
    relationshipName: RelName,
    /**
     * True if we're creating the parent object, instead of updating it. Adding limits 
     * the types of Prisma operations we can perform
     */
    isAdd: IsAdd,
    /**
     * True if object can be transferred to another parent object. Some cases where this is false are 
     * emails, phone numbers, etc.
     */
    isTransferable?: boolean,
    /**
     * True if relationship is one-to-one
     */
    isOneToOne?: IsOneToOne,
    /**
     * True if relationship must be provided
     */
    isRequired?: IsRequired,
    /**
     * Fields to exclude from the relationship data. This can be handled in the shape fields as well,
     * but this is more convenient
     */
    fieldExcludes?: string[],
    /**
     * The Prisma client
     */
    prisma: PrismaType,
    /**
     * Prisma operations to exclude from the relationship data. "isAdd" is often sufficient 
     * to determine which operations to exclude, sometimes it's also nice to exluce "connect" and 
     * "disconnect" when we know that a relationship can only be applied to the parent object
     */
    relExcludes?: RelationshipTypes[],
    /**
     * True if object should be soft-deleted instead of hard-deleted. TODO THIS DOES NOT WORK YET
     */
    softDelete?: boolean,
    /**
     * The name of the ID field. Defaults to "id"
     */
    idField?: IDField,
    /**
     * Functions that perform additional formatting on create and update data. This is often 
     * used to shape grandchildren, great-grandchildren, etc.
     */
    shape?: Mutater<any, any, any, { graphql: any, db: any }, { graphql: any, db: any }>['shape'];
    /**
     * Session data of the user performing the operation. Relationship building is only used when performing 
     * create, update, and delete operations, so id is always required
     */
    userData: SessionUser,
    /**
     * If relationship is a join table, data required to create the join table record
     * 
     * NOTE: Does not differentiate between a disconnect and a delete. How these are handled is 
     * determined by the database cascading.
     */
    joinData?: {
        fieldName: string, // e.g. organization.tags.tag => 'tag'
        uniqueFieldName: string, // e.g. organization.tags.tag => 'organization_tags_taggedid_tagTag_unique'
        childIdFieldName: string, // e.g. organization.tags.tag => 'tagTag'
        parentIdFieldName: string, // e.g. organization.tags.tag => 'taggedId'
        parentId: string | null, // Only needed if not a create
    }
}

type RelConnect<IDField extends string> = { [key in IDField]: string }
type RelDisconnect<IDField extends string> = { [key in IDField]: string }
type RelCreate<Shaped extends { [x: string]: any }> = Shaped
type RelUpdate<Shaped extends { [x: string]: any }, IDField extends string> = { where: { [key in IDField]: string }, data: Shaped }
type RelDelete<IDField extends string> = { [key in IDField]: string }

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
    objectType: GraphQLModelType,
    userData: SessionUser | null | undefined,
    visibility?: VisibilityType | null | undefined,
}