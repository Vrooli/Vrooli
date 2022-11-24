import { GraphQLResolveInfo } from "graphql";
import { Count, PageInfo, SessionUser, TimeFrame } from "../schema/types";
import { PrismaDelegate, PrismaType, PromiseOrValue, RecursivePartial, ReplaceTypes, SingleOrArray } from "../types";
import { ObjectSchema } from 'yup';
import { prisma, Prisma } from "@prisma/client";
import { BuiltRelationship } from "./builder";

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
    'NodeLink' |
    'NodeLinkWhen' |
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
    format: Formatter<GraphQLModel, PermissionObject>;
    display: Displayer;
    delegate: (prisma: PrismaType) => PrismaDelegate;
    search?: Searcher<SearchInput>;
    mutate?: Mutater<GraphQLModel>;
    permissions?: () => Permissioner<PermissionObject, SearchInput>;
    validate?: Validator<GraphQLModel, PermissionsQuery>
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
        languages: string[],
        objects: ({ id: string } & { [x: string]: any })[], // TODO: fix this type
        partial: PartialGraphQLInfo,
        prisma: PrismaType,
        userData: SessionUser | null,
    }) => [GQLFields, () => any][];
}

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export interface Formatter<GraphQLModel, GQLFields extends string> {
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
    permissionsSelect: (userId: string | null, languages: string[]) => PermissionsSelect;
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
     * Uses query result to determine if the object is soft-deleted
     */
    isDeleted: (data: PrismaObject, languages: string[]) => boolean;
    /**
     * Uses query result to determine if the object is public. This typically means "isPrivate" and "isDeleted" are false
     */
    isPublic: (data: PrismaObject, languages: string[]) => boolean;
    /**
     * Permissions data for the object's owner
     */
    owner: (data: PrismaObject) => {
        Organization?: { [x: string]: any } | null;
        User?: { [x: string]: any } | null;
    }
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
        connect?: ({ connectMany, languages, prisma, userId }: {
            connectMany: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        create?: ({ createMany, deltaAdding, languages, prisma, userId }: {
            createMany: GQLCreate[],
            deltaAdding: number,
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        delete?: ({ deleteMany, languages, prisma, userId }: {
            deleteMany: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        disconnect?: ({ disconnectMany, languages, prisma, userId }: {
            disconnectMany: string[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => Promise<void> | void;
        update?: ({ languages, prisma, updateMany, userId }: {
            prisma: PrismaType,
            updateMany: GQLUpdate[],
            userData: SessionUser,
        }) => Promise<void> | void;
    };
    /**
     * Any custom transformations you want to perform before a create/update mutation, 
     * besides the ones supported by default in cudHelper. This includes converting creates to 
     * connects, which means this function has to be pretty flexible in what it allows
     */
    transformations?: {
        create?: (createMany: GQLCreate[], userId: string) => Promise<GQLCreate[]> | GQLCreate[];
        update?: (updateMany: GQLUpdate[], userId: string) => Promise<GQLUpdate[]> | GQLUpdate[];
    };
};

/**
 * Describes shape of component that can be duplicated
 */
export type Duplicater<
    // Select must include "id" and "intendToPullRequest" fields
    Select extends { id?: boolean | undefined, intendToPullRequest?: boolean | undefined, [x: string]: any },
    Data extends { [x: string]: any }
> = {
    /**
     * Data to select from the database to duplicate. DO NOT select anything that 
     * is not meant to be duplicated 
     * 
     * 
     * NOTE: All IDs will be converted to new IDs. This is especially useful for 
     * child data that references each other, like nodes and edges 
     */
    select: Select;
    /**
     * Data to connect to new owner
     */
    owner: (id: string) => {
        Organization?: Partial<Data> | null
        User?: Partial<Data> | null;
    }
}

export type MutaterShapes = {
    graphql: { [x: string]: any },
    db: { [x: string]: any },
}

export type RelBuilderInput<RelName extends string, Relationship extends MutaterShapes> = {
    data: { [RelName]: Relationship['graphql'] },
    prisma: PrismaType,
    relationshipName: RelName,
    userData: SessionUser,
}

/**
 * Describes shape of component that can be mutated
 */
export type Mutater<
    GQLObject extends { [x: string]: any },
    Create extends MutaterShapes | false,
    Update extends MutaterShapes | false,
    RelationshipCreate extends MutaterShapes | false,
    RelationshipUpdate extends MutaterShapes | false,
> = {
    /**
     * Shapes data for create/update mutations, both as a main 
     * object and as a relationship object
     */
    shape: (Create extends false ? {} : {
        create: ({ data, prisma, userData }: {
            data: Create['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<Create['db']>,
    }) & (Update extends false ? {} : {
        update: ({ data, prisma, userData }: {
            data: Update['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<Update['db']>,
    }) & (RelationshipCreate extends false ? {} : {
        relCreate: ({ data, prisma, userData }: {
            data: RelationshipCreate['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<RelationshipCreate['db']>,
    }) & (RelationshipUpdate extends false ? {} : {
        relUpdate: ({ data, prisma, userData }: {
            data: RelationshipUpdate['graphql'],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<RelationshipUpdate['db']>,
    }),
    /**
     * Triggers when a mutation is performed on the object
     */
    trigger?: (Create extends false ? {} : {
        onCreated?: ({ created, prisma, userData }: {
            created: RecursivePartial<GQLObject>[],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    }) & (Update extends false ? {} : {
        onUpdated?: ({ updated, updateInput, prisma, userData }: {
            updated: RecursivePartial<GQLObject>[],
            updateInput: Update['graphql'][],
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    }) & {
        onDeleted?: ({ deleted, prisma, userData }: {
            deleted: Count,
            prisma: PrismaType,
            userData: SessionUser,
        }) => PromiseOrValue<void>,
    },
    yup: (Create extends false ? {} : {
        create: ObjectSchema,
    }) & (Update extends false ? {} : {
        update: ObjectSchema,
    });
}

/**
 * Functions for displaying an object
 */
export type Displayer = {
    /**
     * Finds a string to represent each object in each user's preferred language
     * @param prisma Prisma client
     * @param objects List of object ids and the corresponding user's preferred languages
     * @returns List of strings to represent each object in each user's preferred language, 
     * in the same order as the input
     */
    labels: (prisma: PrismaType, objects: { id: string, languages: string[] }[]) => PromiseOrValue<string[]>,
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

export interface CUDResult<GraphQLObject> {
    created?: RecursivePartial<GraphQLObject>[],
    updated?: RecursivePartial<GraphQLObject>[],
    deleted?: Count, // Number of deleted organizations
}

export interface CUDHelperInput<
    GQLObject extends { [x: string]: any }
> {
    createMany?: { [x: string]: any }[] | null | undefined;
    deleteMany?: string[] | null | undefined,
    objectType: GraphQLModelType,
    partialInfo: PartialGraphQLInfo,
    prisma: PrismaType,
    updateMany?: { [x: string]: any }[] | null | undefined,
    userData: SessionUser,
}