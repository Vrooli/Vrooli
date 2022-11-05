import { GraphQLResolveInfo } from "graphql";
import { Count, PageInfo, TimeFrame } from "../schema/types";
import { PrismaType, RecursivePartial } from "../types";

export enum GraphQLModelType {
    // Api = 'Api',
    Comment = 'Comment',
    Copy = 'Copy',
    DevelopPageResult = 'DevelopPageResult',
    Email = 'Email',
    Fork = 'Fork',
    Handle = 'Handle',
    HistoryPageResult = 'HistoryPageResult',
    HomePageResult = 'HomePageResult',
    InputItem = 'InputItem',
    LearnPageResult = 'LearnPageResult',
    Member = 'Member',
    Node = 'Node',
    NodeEnd = 'NodeEnd',
    NodeLoop = 'NodeLoop',
    NodeRoutineList = 'NodeRoutineList',
    NodeRoutineListItem = 'NodeRoutineListItem',
    Organization = 'Organization',
    OutputItem = 'OutputItem',
    Profile = 'Profile',
    Project = 'Project',
    ProjectOrRoutineSearchResult = 'ProjectOrRoutineSearchResult',
    ProjectOrOrganizationSearchResult = 'ProjectOrOrganizationSearchResult',
    Report = 'Report',
    ResearchPageResult = 'ResearchPageResult',
    Resource = 'Resource',
    ResourceList = 'ResourceList',
    Role = 'Role',
    Routine = 'Routine',
    Run = 'Run',
    RunInput = 'RunInput',
    RunStep = 'RunStep',
    Standard = 'Standard',
    Star = 'Star',
    Tag = 'Tag',
    TagHidden = 'TagHidden',
    User = 'User',
    View = 'View',
    Vote = 'Vote',
    Wallet = 'Wallet',
}

/**
 * Basic structure of an object's business layer.
 * Every business layer object has at least a PrismaType object and a format converter. 
 * Everything else is optional
 */
 export type ModelLogic<GraphQLModel, SearchInput, PermissionObject> = {
    format: FormatConverter<GraphQLModel, PermissionObject>;
    prismaObject: (prisma: PrismaType) => PrismaType[keyof PrismaType];
    search?: Searcher<SearchInput>;
    mutate?: (prisma: PrismaType) => Mutater<GraphQLModel>;
    permissions?: () => Permissioner<PermissionObject, SearchInput>;
    verify?: { [x: string]: any };
    query?: (prisma: PrismaType) => Querier;
    type: keyof typeof GraphQLModelType;
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
    [x: string]: keyof typeof GraphQLModelType | undefined | boolean | { [x: string]: PartialGraphQLInfo };
    __typename?: keyof typeof GraphQLModelType;
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

type NestedGraphQLModelType = keyof typeof GraphQLModelType | { [fieldName: string]: NestedGraphQLModelType };

export type RelationshipMap<GraphQLModel> = { [key in keyof GraphQLModel]?: NestedGraphQLModelType } & { __typename: keyof typeof GraphQLModelType };

export type UnionMap<GraphQLModel> = { [key in keyof GraphQLModel]?: { [x: string]: string } };

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export type FormatConverter<GraphQLModel, PermissionObject> = {
    /**
     * Maps relationship names to their GraphQL type. 
     * If the relationship is a union (i.e. has mutliple possible types), 
     * the GraphQL type will be an object of field/GraphQLModelType pairs.
     */
    relationshipMap: RelationshipMap<GraphQLModel>;
    /**
     * Maps GraphQL union fields to their corresponding Prisma fields. 
     * Each field is a key from the GraphQLModel type
     */
    unionMap?: UnionMap<GraphQLModel>;
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

export interface ValidateMutationsInput<Create, Update> {
    userId: string | null,
    createMany?: Create[] | null | undefined,
    updateMany?: { where: { [x: string]: any }, data: Update }[] | null | undefined,
    deleteMany?: string[] | null | undefined,
}

export interface CUDInput<Create, Update> {
    userId: string | null,
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