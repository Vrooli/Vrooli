import { GraphQLResolveInfo } from "graphql";
import { Count, PageInfo, SessionUser, TimeFrame } from "../schema/types";
import { PrismaDelegate, PrismaType, RecursivePartial, ReplaceTypes, SingleOrArray } from "../types";
import { ObjectSchema } from 'yup';
import { prisma, Prisma } from "@prisma/client";

export type GraphQLModelType =
    'Comment' |
    'Copy' |
    'DevelopPageResult' |
    'Email' |
    'Fork' |
    'Handle' |
    'HistoryPageResult' |
    'HomePageResult' |
    'InputItem' |
    'LearnPageResult' |
    'Member' |
    'Node' |
    'NodeEnd' |
    'NodeLoop' |
    'NodeRoutineList' |
    'NodeRoutineListItem' |
    'Organization' |
    'OutputItem' |
    'Profile' |
    'Project' |
    'ProjectOrRoutineSearchResult' |
    'ProjectOrOrganizationSearchResult' |
    'Report' |
    'ResearchPageResult' |
    'Resource' |
    'ResourceList' |
    'Role' |
    'Routine' |
    'RunRoutine' |
    'RunInput' |
    'RunStep' |
    'Standard' |
    'Star' |
    'Tag' |
    'TagHidden' |
    'User' |
    'View' |
    'Vote' |
    'Wallet';

/**
 * Allows Prisma select fields to map to GraphQLModelTypes. Any field which can be 
 * an object (e.g. a relation) should be able to specify either a GraphQLModelType or
 * a nested ValidateMap
 */
export type ValidateMap<T> = {
    // TODO allows too many fields. Should only allow relationships
    [K in keyof T]?: GraphQLModelType | ValidateMap<T[K]>
};

/**
 * Recursively pads object with "select" fields
 */
export type WithSelect<T> = { select: { [K in keyof T]: T[K] extends object ? WithSelect<T[K]> : true } };

/**
 * Basic structure of an object's business layer.
 * Every business layer object has at least a PrismaType object and a format converter. 
 * Everything else is optional
 */
export type ModelLogic<GraphQLModel, SearchInput, PermissionObject, PermissionsQuery> = {
    format: FormatConverter<GraphQLModel, PermissionObject>;
    prismaObject: (prisma: PrismaType) => PrismaDelegate;
    search?: Searcher<SearchInput>;
    mutate?: (prisma: PrismaType) => Mutater<GraphQLModel>;
    permissions?: () => Permissioner<PermissionObject, SearchInput>;
    validate?: Validator<GraphQLModel, PermissionsQuery>
    query?: (prisma: PrismaType) => Querier;
    type: GraphQLModelType;
}

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
export type PartialPrismaSelect = { __typename?: GraphQLModelType, [x: string]: boolean | PartialPrismaSelect };

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
    }) => Promise<{ count: number }>;
}

export type NestedGraphQLModelType = GraphQLModelType | { [fieldName: string]: NestedGraphQLModelType } | { root: GraphQLModelType } | { root: NestedGraphQLModelType };

export type RelationshipMap<GraphQLModel> = { [key in keyof GraphQLModel]?: NestedGraphQLModelType } & { __typename: GraphQLModelType };

/**
 * Helper functions for adding and removing supplemental fields. These are fields 
 * are requested in the select query, but are either not in the main database or 
 * cannot be requested in the same query (e.g. isStarred, permissions) 
 */
export interface SupplementalConverter<GQLModel, GQLFields extends string> {
    /**
     * List of all supplemental fields added to the GraphQL model after the main query 
     * (i.e. all fields to be excluded)
     */
    graphqlFields: GQLFields[];
    /**
     * List of all fields to add to the Prisma select query, in order to calculate 
     * supplemental fields
     */
    dbFields?: string[]; // TODO make type safer
    /**
     * An array of resolver functions, one for each calculated field
     */
    toGraphQL: ({ ids, objects, partial, prisma, userId }: {
        ids: string[],
        objects: ({ id: string } & { [x: string]: any })[], // TODO: fix this type
        partial: PartialGraphQLInfo,
        prisma: PrismaType,
        userData: SessionUser | null,
    }) => [GQLFields, () => any][];
}

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export interface FormatConverter<GraphQLModel, GQLFields extends string> {
    /**
     * Maps relationship names to their GraphQL type. 
     * If the relationship is a union (i.e. has mutliple possible types), 
     * the GraphQL type will be an object of field/GraphQLModelType pairs. 
     * NOTE: The keyword "root" is used to indicate that a relationship belongs in the root 
     * object. This only applies when working with versioned data
     */
    relationshipMap: RelationshipMap<GraphQLModel>;
    /**
     * Maps primitive fields in a versioned object's root table to the GraphQL type. 
     */
    rootFields?: string[];
    /**
     * Add join tables which are not present in GraphQL object
     */
    addJoinTables?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Remove join tables which are not present in GraphQL object
     */
    removeJoinTables?: (data: { [x: string]: any }) => any;
    /**
     * Add _count fields
     */
    addCountFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Remove _count fields
     */
    removeCountFields?: (data: { [x: string]: any }) => any;
    /**
     * Data for adding supplemental fields to the GraphQL object
     */
    supplemental?: SupplementalConverter<GraphQLModel, GQLFields>;
}


/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<SearchInput, SortBy extends string, OrderBy extends { [x: string]: any }, Where extends { [x: string]: any }> = {
    defaultSort: SortBy;
    sortMap: { [key in SortBy]: OrderBy };
    searchStringQuery: ({ insensitive, languages, searchString }: {
        insensitive: { contains: string; mode: 'default' | 'insensitive'; },
        languages?: string[],
        searchString: string,
    }) => Where;
    customQueries?: (input: SearchInput, userId: string | null | undefined) => Where;
}

/**
 * Describes shape of component that can be queried
 */
export type Querier = { [x: string]: any };

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    GQLModel extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends { [x: string]: any },
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any }
> = {
    /**
     * Maps relationsips on the object's database schema to the corresponding GraphQL type,
     * if they require validation
     * 
     * Examples include: 
     * routine -> organization
     * node -> routine -> organization
     * createdByOrganizationId
     * 
     * Examples when this is not needed:
     * project -> resourceList
     */
    validateMap: { __typename: GraphQLModelType } & ValidateMap<PermissionsSelect>;
    /**
     * Select query to calculate the object's permissions. This will be used - possibly in 
     * conjunction with the parent object's permissions (also queried in this field) - to determine if you 
     * are allowed to perform the mutation
     */
    permissionsSelect: (userId: string | null) => PermissionsSelect;
    /**
     * Array of resolvers to calculate the object's permissions
     */
    permissionResolvers: ({ isAdmin, isDeleted, isPublic }: {
        isAdmin: boolean,
        isDeleted: boolean,
        isPublic: boolean,
    }) => [keyof Omit<PermissionObject, '__typename'>, () => any][];
    /**
     * Partial where query added to the object's search query. Useful when querying things like an organization's routines, 
     * where you should only see private routines if you are a member of the organization
     */
    ownerOrMemberWhere: (userId: string) => OwnerOrMemberWhere;
    /**
     * Uses query result to determine if the user has admin/owner privileges for the object
     */
    isAdmin: (data: PrismaObject, userId: string) => boolean;
    /**
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: PrismaObject) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false
     */
    isPublic: (data: PrismaObject) => boolean;
    /**
     * String fields which must be checked for profanity. You don't need to 
     * include fields in a translate object, as those will be checked
     * automatically
     */
    profanityFields?: string[];
    /**
     * Any custom validations you want to perform before a create mutation. You must throw 
     * an error if a validation fails, since that'll return a customized error message to the
     * user
     */
    validations?: {
        connect?: (connectMany: string[], prisma: PrismaType, userId: string) => Promise<void> | void;
        create?: (createMany: GQLCreate[], prisma: PrismaType, userId: string, deltaAdding: number) => Promise<void> | void;
        delete?: (deleteMany: string[], prisma: PrismaType, userId: string) => Promise<void> | void;
        disconnect?: (disconnectMany: string[], prisma: PrismaType, userId: string) => Promise<void> | void;
        update?: (updateMany: GQLUpdate[], prisma: PrismaType, userId: string) => Promise<void> | void;
    };
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones specified in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations?: {
        create?: (createMany: GQLCreate[], userId: string) => Promise<GQLCreate[]> | GQLCreate[];
        update?: (updateMany: GQLUpdate[], userId: string) => Promise<GQLUpdate[]> | GQLUpdate[];
    };
};

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<GQLModel> = {
    relationshipBuilder?(userId: string, data: { [x: string]: any }, isAdd: boolean = true, relationshipName?: string | undefined): Promise<{ [x: string]: any } | undefined>;
    cud?({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<any, any>): Promise<CUDResult<GQLModel>>;
    duplicate?({ userId, objectId, isFork, createCount }: DuplicateInput): Promise<DuplicateResult<GQLModel>>;
} & { [x: string]: any };

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };

/**
 * Mapper for associating a model's GraphQL count fields to the relationships they count
 */
export type CountMap = { [key: string]: string };

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

export interface CUDInput<Create, Update> {
    userData: SessionUser,
    createMany?: Create[] | null | undefined,
    updateMany?: { where: { [x: string]: any }, data: Update }[] | null | undefined,
    deleteMany?: string[] | null | undefined,
    partialInfo: PartialGraphQLInfo,
}

export interface CUDResult<GraphQLObject> {
    created?: RecursivePartial<GraphQLObject>[],
    updated?: RecursivePartial<GraphQLObject>[],
    deleted?: Count, // Number of deleted organizations
}

export interface CUDHelperInput<GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }, GraphQLObject, DBCreate extends { [x: string]: any }, DBUpdate extends { [x: string]: any }> {
    objectType: GraphQLModelType,
    userData: SessionUser,
    prisma: PrismaType,
    createMany?: GraphQLCreate[] | null | undefined,
    updateMany?: { where: { [x: string]: any }, data: GraphQLUpdate }[] | null | undefined,
    deleteMany?: string[] | null | undefined,
    partialInfo: PartialGraphQLInfo,
    yup: { yupCreate: ObjectSchema, yupUpdate: ObjectSchema },
    shape: {
        shapeCreate: (userId: string, create: GraphQLCreate) => (Promise<DBCreate> | DBCreate),
        shapeUpdate: (userId: string, update: GraphQLUpdate) => (Promise<DBUpdate> | DBUpdate),
    },
    onCreated?: (created: RecursivePartial<GraphQLObject>[]) => Promise<void> | void,
    onUpdated?: (updated: RecursivePartial<GraphQLObject>[], updateData: GraphQLUpdate[]) => Promise<void> | void,
    onDeleted?: (deleted: Count) => Promise<void> | void,
}

export interface DuplicateInput {
    /**
     * The userId of the user making the copy
     */
    userId: string,
    /**
     * The id of the object to copy.
     */
    objectId: string,
    /**
     * Whether the copy is a fork or a copy
     */
    isFork: boolean,
    /**
     * Number of child objects already created. Can be used to limit size of copy.
     */
    createCount: number,
}

export interface DuplicateResult<GraphQLObject> {
    object: RecursivePartial<GraphQLObject>,
    numCreated: number
}