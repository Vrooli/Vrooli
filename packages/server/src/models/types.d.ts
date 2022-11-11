import { GraphQLResolveInfo } from "graphql";
import { Count, PageInfo, TimeFrame } from "../schema/types";
import { PrismaDelegate, PrismaType, RecursivePartial } from "../types";
import { ObjectSchema } from 'yup';
import { Prisma } from "@prisma/client";

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
    'Routine' | 'Run' |
    'RunInput' |
    'RunStep' |
    'Standard' |
    'Star' | 'Tag' |
    'TagHidden' |
    'User' |
    'View' |
    'Vote' |
    'Wallet';

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
    [x: string]: GraphQLModelType | undefined | boolean | { [x: string]: PartialGraphQLInfo };
    __typename?: GraphQLModelType;
}

/**
 * Shape 3 of 4 for GraphQL to Prisma conversion. Still contains the __typename fields, 
 * but does not pad objects with a "select" field. Calculated fields, join tables, and other 
 * data transformations from the GraphqL shape are removed. This is useful when checking 
 * which fields are requested from a Prisma query.
 */
export type PartialPrismaSelect = { [x: string]: any };

/**
 * Shape 4 of 4 for GraphQL to Prisma conversion. This is the final shape of the requested data 
 * as it will be sent to the database. It is has __typename fields removed, and objects padded with "select"
 */
export type PrismaSelect = { [x: string]: any };

export type NestedGraphQLModelType = GraphQLModelType | { [fieldName: string]: NestedGraphQLModelType } | { root: GraphQLModelType } | { root: NestedGraphQLModelType };

export type RelationshipMap<GraphQLModel> = { [key in keyof GraphQLModel]?: NestedGraphQLModelType } & { __typename: GraphQLModelType };

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export type FormatConverter<GraphQLModel, PermissionObject> = {
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
     * Removes fields which are not in the database (i.e. calculated/supplemental fields)
     */
    removeSupplementalFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Adds fields which are calculated after the main query
     * @returns objects ready to be sent through GraphQL
     */
    addSupplementalFields?: ({ objects, partial, permissions, prisma, userId }: {
        objects: ({ id: string } & { [x: string]: any })[];
        partial: PartialGraphQLInfo,
        permissions?: PermissionObject[] | null,
        prisma: PrismaType,
        userId: string | null,
    }) => Promise<RecursivePartial<GraphQLModel>[]>;
}


/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<SearchInput> = {
    defaultSort: any;
    getSortQuery: (sortBy: string) => any;
    getSearchStringQuery: (searchString: string, languages?: string[]) => any;
    customQueries?: (input: SearchInput, userId: string | null | undefined) => { [x: string]: any };
}

/**
 * Describes shape of component that can be queried
 */
export type Querier = { [x: string]: any };

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<GraphQLModel extends { [x: string]: any }, PermissionsQuery extends { [x: string]: any }> = {
    /**
     * Relationships in the GraphQL object which have separate/additional 
     * validation rules
     */
    validatedRelationshipMap?: { [key in keyof GraphQLModel]?: GraphQLModelType };
    /**
     * Query to get the object's permissions. This will be used - possibly in 
     * conjunction with the parent object's permissions - to determine if you 
     * are allowed to perform the mutation
     */
    permissionsQuery?: (userId: string) => PermissionsQuery;
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
        connect?: (connectMany: string[], userId: string) => Promise<void> | void;
        create?: (createMany: GraphQLCreate[], userId: string) => Promise<void> | void;
        delete?: (deleteMany: string[], userId: string) => Promise<void> | void;
        disconnect?: (disconnectMany: string[], userId: string) => Promise<void> | void;
        update?: (updateMany: GraphQLUpdate[], userId: string) => Promise<void> | void;
    };
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones specified in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations: {
        create: (createMany: GraphQLCreate[], userId: string) => Promise<GraphQLCreate[]> | GraphQLCreate[];
        update: (updateMany: GraphQLUpdate[], userId: string) => Promise<GraphQLUpdate[]> | GraphQLUpdate[];
    };
} & { [x: string]: any };

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<GraphQLModel> = {
    cud?({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<any, any>): Promise<CUDResult<GraphQLModel>>;
    duplicate?({ userId, objectId, isFork, createCount }: DuplicateInput): Promise<DuplicateResult<GraphQLModel>>;
} & { [x: string]: any };

/**
 * Describes shape of component with permissioned access
 */
export type Permissioner<PermissionObject, SearchInput> = {
    /**
     * Permissions for the object
     */
    get({ objects, permissions, prisma, userId }: {
        objects: ({ id: string } & { [x: string]: any })[],
        permissions?: PermissionObject[] | null,
        prisma: PrismaType,
        userId: string | null,
    }): Promise<PermissionObject[]>
    /**
     * Checks if user has permissions to complete search input, and if search can include 
     * private objects
     * @returns 'full' if user has permissions and search can include private objects, 
     * 'public' if user has permissions and search cannot include private objects, 
     * 'none' if user does not have permission to search 
     */
    canSearch?({ input, userId }: {
        input: SearchInput,
        prisma: PrismaType,
        userId: string | null,
    }): Promise<'full' | 'public' | 'none'>
    /**
     * Query format for checking ownership of an object
     * @param userId - ID to check ownership against
     * @returns Prisma where clause for checking ownership
     */
    ownershipQuery(userId: string): { [x: string]: any }
}

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
    userId: string,
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
    userId: string,
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