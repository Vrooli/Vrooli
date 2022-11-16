import { GraphQLResolveInfo } from "graphql";
import { Count, PageInfo, TimeFrame } from "../schema/types";
import { PrismaDelegate, PrismaType, RecursivePartial } from "../types";
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
export type PrismaSelect = { [x: string]: any };

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
        userId: string | null,
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
 * Base permissions that any object can have
 */
export type BasePermissions = {
    canDelete: boolean;
    canReport: boolean;
    canEdit: boolean;
    canView: boolean;
}

/**
 * Describes shape of component that has validation rules 
 */
export type Validator<
    GQLCreate extends { [x: string]: any },
    GQLUpdate extends { [x: string]: any },
    GQLModel extends { [x: string]: any },
    PrismaObject extends { [x: string]: any },
    PermissionObject extends BasePermissions,
    PermissionsSelect extends { [x: string]: any },
    OwnerOrMemberWhere extends { [x: string]: any }
> = {
    /**
     * Relationships in the GQLCreate, GQLUpdate, or GQLModel object which have separate/additional 
     * validation rules. Typically any selectable relationship which can lead to a different owner. 
     * 
     * Examples include: 
     * routine -> organization
     * node -> routine -> organization
     * createdByOrganizationId
     * 
     * Examples when this is not needed:
     * project -> resourceList
     * 
     * NOTE: Will automatically handle checking fields with the "Id", "Create", "Update", or "Delete" suffix, 
     * as well as deconstructing unions (e.g. 'creator': { 'user': 'User', 'organization': 'Organization' } will check 'creator', 'user', 'userId', etc.)
     */
    validateMap: { [key in keyof (GQLCreate & GQLUpdate & PartialPrismaSelect)]?: GraphQLModelType | { [x: string]: GraphQLModelType } } & { __typename: GraphQLModelType };
    /**
     * Select query to calculate the object's permissions. This will be used - possibly in 
     * conjunction with the parent object's permissions (also queried in this field) - to determine if you 
     * are allowed to perform the mutation
     */
    permissionsSelect: PermissionsSelect;
    /**
     * Array of resolvers to calculate the object's permissions
     */
    permissionResolvers: (data: PrismaObject, userId: string | null) => [keyof Exclude<PermissionObject, '__typename'>, () => any][];
    /**
     * Partial where query added to the object's search query. Useful when querying things like an organization's routines, 
     * where you should only see private routines if you are a member of the organization
     */
    ownerOrMemberWhere: (userId: string) => OwnerOrMemberWhere;
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