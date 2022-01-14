// Components for providing basic functionality to model objects
import { CODE } from '@local/shared';
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, PageInfo, InputMaybe, ReportInput, Scalars, TimeFrame } from '../schema/types';
import { CustomError } from '../error';
import { PrismaType, RecursivePartial } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
import graphqlFields from 'graphql-fields';


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Required fields to update any object
interface UpdateInterface {
    id?: Scalars['ID'] | InputMaybe<string>;
}

/**
 * Describes shape of component that converts between Prisma and GraphQL object types.
 * This is often used for removing the extra nesting caused by joining tables
 * (e.g. User -> UserRole -> Role becomes UserRole -> Role)
 */
export type FormatConverter<GraphQLModel, FullDBModel> = {
    /**
     * Helps add/remove many-to-many join tables, since they are not included in GraphQL
     */
    joinMapper?: JoinMap;
    /**
     * Helps add/remove fields for relationship counts, since they are not included formatted the same in GraphQL vs. Prisma
     */
    countMapper?: CountMap;
    /**
     * Converts object from GraphQL representation to Prisma
     */
    toDB: (obj: RecursivePartial<GraphQLModel>) => RecursivePartial<FullDBModel>;
    /**
     * Converts object from Prisma representation to GraphQL
     */
    toGraphQL: (obj: RecursivePartial<FullDBModel>) => RecursivePartial<GraphQLModel>;
}

/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Sortable<SortBy> = {
    defaultSort: any;
    getSortQuery: (sortBy: string) => any;
    getSearchStringQuery: (searchString: string) => any;
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

type BaseType = PrismaModels['comment']; // It doesn't matter what PrismaType is used here, it's just to help TypeScript handle Prisma operations

// Strings for accessing model functions from Prisma
export const MODEL_TYPES = {
    Comment: 'comment',
    Email: 'email',
    Node: 'node',
    Organization: 'organization',
    Project: 'project',
    Resource: 'resource',
    Role: 'role',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user'
} as const

// Types for Prisma model objects. These are required when indexing the Prisma model without using dot notation
type Models<T> = {
    [MODEL_TYPES.Comment]: Prisma.commentDelegate<T>;
    [MODEL_TYPES.Email]: Prisma.emailDelegate<T>;
    [MODEL_TYPES.Node]: Prisma.nodeDelegate<T>;
    [MODEL_TYPES.Organization]: Prisma.organizationDelegate<T>;
    [MODEL_TYPES.Project]: Prisma.projectDelegate<T>;
    [MODEL_TYPES.Resource]: Prisma.resourceDelegate<T>;
    [MODEL_TYPES.Role]: Prisma.roleDelegate<T>;
    [MODEL_TYPES.Routine]: Prisma.routineDelegate<T>;
    [MODEL_TYPES.Standard]: Prisma.standardDelegate<T>;
    [MODEL_TYPES.Tag]: Prisma.tagDelegate<T>;
    [MODEL_TYPES.User]: Prisma.userDelegate<T>;
}
export type PrismaModels = Models<Prisma.RejectOnNotFound | Prisma.RejectPerOperation | undefined>

export type InfoType = GraphQLResolveInfo | { [x: string]: any } | null;

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

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

/**
 * Helper function for adding join tables between 
 * many-to-many relationship parents and children
 * @param obj - GraphQL-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const addJoinTables = (obj: any, map: JoinMap): any => {
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        if (obj[key]) result[key] = ({ [value]: obj[key] })
    }
    return {
        ...obj,
        ...result
    }
}

/**
 * Helper function for converting GraphQL count fields to Prisma relationship counts
 * @param obj - GraphQL-shaped object
 * @param map - Mapping of GraphQL field names to Prisma relationship names
 */
 export const addCountQueries = (obj: any, map: CountMap): any => {
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        if (obj[key]) {
            if (!obj._count) obj._count = {};
            obj._count[value] = true;
            delete obj[key];
        }
    }
    return {
        ...obj,
        ...result
    }
}

/**
 * Helper function for removing join tables between
 * many-to-many relationship parents and children
 * @param obj - DB-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const removeJoinTables = (obj: any, map: JoinMap): any => {
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        if (obj[key] && obj[key][value]) result[key] = obj[key][value];
    }
    return {
        ...obj,
        ...result
    }
}

/**
 * Helper function for converting Prisma relationship counts to GraphQL count fields
 * @param obj - Prisma-shaped object
 * @param map - Mapping of GraphQL field names to Prisma relationship names
 */
 export const removeCountQueries = (obj: any, map: CountMap): any => {
    // Create result object
    let result: any = {};
    // If no counts, no reason to continue
    if (!obj._count) return obj;
    // Iterate over count map
    for (const [key, value] of Object.entries(map)) {
        if (obj._count[value] !== undefined && obj._count[value] !== null) {
            obj[key] = obj._count[value];
        }
    }
    // Make sure to delete _count field
    delete obj._count;
    return {
        ...obj,
        ...result
    }
}

/**
 * Converts the {} values of a graphqlFields object to true
 */
const formatGraphQLFields = (fields: { [x: string]: any }): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length === 0) converted[key] = true;
        else converted[key] = formatGraphQLFields(fields[key]);
    });
    return converted;
}

/**
 * Adds "select" to the correct of an object to make it a Prisma select
 */
const padSelect = (fields: { [x: string]: any }): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length > 0) converted[key] = padSelect(fields[key]);
        else converted[key] = true;
    });
    return { select: converted };
}

/**
 * Helper function for creating a Prisma select object. 
 * If the select object is in the shape of a paginated search query, 
 * then it will be converted to a prisma select object.
 * @returns select object for Prisma operations
 */
export const selectHelper = <GraphQLModel, FullDBModel>(info: InfoType, toDB: FormatConverter<GraphQLModel, FullDBModel>['toDB']): any => {
    // Return undefined if info not set
    if (!info) return undefined;
    // Find select fields in info object
    let select = info.hasOwnProperty('fieldName') ?
        formatGraphQLFields(graphqlFields((info as GraphQLResolveInfo), {}, {})) :
        info;
    // If fields are in the shape of a paginated search query, then convert to a Prisma select object
    if (select.hasOwnProperty('pageInfo') && select.hasOwnProperty('edges')) {
        select = select.edges.node;
    }
    // Convert select from graphQL to database
    select = toDB(select as any);
    // Make sure to delete __typename field
    delete select.__typename;
    return padSelect(select);
}

/**
 * Compositional component for models which can be queried by ID
 * @param state 
 * @returns 
 */
export const findByIder = <GraphQLModel, FullDBModel>(model: keyof PrismaType, toDB: FormatConverter<GraphQLModel, FullDBModel>['toDB'], prisma?: PrismaType) => ({
    async findById(input: FindByIdInput, info: InfoType): Promise<RecursivePartial<FullDBModel> | null> {
        console.log('FIND BY ID', Boolean(prisma), input)
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!input.id) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
        // Access database
        return await (prisma[model] as BaseType).findUnique({ where: { id: input.id }, ...select }) as unknown as Partial<FullDBModel> | null;
    }
})

/**
 * Compositional component for models which can be created directly from an input
 * NOTE: This is only a basic implementation, and therefore does not handle relationships
 * @param state 
 * @returns 
 */
export const creater = <ModelInput, GraphQLModel, FullDBModel>(model: keyof PrismaType, toDB: FormatConverter<GraphQLModel, FullDBModel>['toDB'], prisma?: PrismaType) => ({
    async create(input: ModelInput, info: InfoType): Promise<RecursivePartial<FullDBModel>> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
        // Access database
        return await (prisma[model] as BaseType).create({ data: { ...input }, ...select }) as unknown as RecursivePartial<FullDBModel>;
    }
})

/**
 * Compositional component for models which can be updated directly from an input
 * @param state 
 * @returns 
 */
export const updater = <ModelInput extends UpdateInterface, GraphQLModel, FullDBModel>(
    model: keyof PrismaType, 
    toDB: FormatConverter<GraphQLModel, FullDBModel>['toDB'], 
    prisma?: PrismaType) => ({
    async update(input: ModelInput, info: InfoType): Promise<RecursivePartial<FullDBModel>> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!input.id) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
        // Access database
        return await (prisma[model] as BaseType).update({ where: { id: input.id }, data: { ...input }, ...select }) as unknown as RecursivePartial<FullDBModel>;
    }
})

/**
 * Compositional component for models which can be deleted directly.
 * NOTE: In most situations, deletes should be wrapped in another function for checking
 * if the delete is allowed.
 * @param state 
 * @returns 
 */
export const deleter = (model: keyof PrismaType, prisma?: PrismaType) => ({
    // Delete a single object
    async delete(input: DeleteOneInput): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!input.id) throw new CustomError(CODE.InvalidArgs);
        // Access database
        return await (prisma[model] as BaseType).delete({ where: { id: input.id } }) as unknown as boolean;
    },
    // Delete many objects
    async deleteMany(input: DeleteManyInput): Promise<Count> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!input.ids) throw new CustomError(CODE.InvalidArgs);
        // Access database
        return await (prisma[model] as BaseType).deleteMany({ where: { id: { in: input.ids } } });
    }
})

/**
 * Compositional component for models which can be reported
 * @param state 
 * @returns 
 */
export const reporter = () => ({
    async report(input: ReportInput): Promise<boolean> {
        if (!Boolean(input.id)) throw new CustomError(CODE.InvalidArgs);
        throw new CustomError(CODE.NotImplemented);
    }
})

/**
 * Compositional component for models which can be searched
 * @param state 
 * @returns 
 */
export const searcher = <SortBy, SearchInput extends SearchInputBase<SortBy>, GraphQLModel, FullDBModel>(
    model: keyof PrismaType, 
    toDB: FormatConverter<GraphQLModel, FullDBModel>['toDB'],
    toGraphQL: FormatConverter<GraphQLModel, FullDBModel>['toGraphQL'],
    sorter: Sortable<any>, 
    prisma?: PrismaType) => ({
    /**
     * Cursor-based search. Supports pagination, sorting, and filtering by string.
     * @param where Additional where clauses to apply to the search
     * @param input GraphQL-provided search parameters
     * @param info Requested return information
     * @returns 
     */
    async search(where: { [x: string]: any }, input: SearchInput, info: InfoType): Promise<PaginatedSearchResult> {
        const boop = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
        console.log('SEARCH BOIII', boop)
        console.log('SEARCH BOIII _COUNT', boop._count)
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
        // Create query for specified ids
        const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
        // Determine sort order
        const sortQuery = sorter.getSortQuery(input.sortBy ?? sorter.defaultSort);
        // Determine text search query
        const searchQuery = input.searchString ? sorter.getSearchStringQuery(input.searchString) : undefined;
        // Determine createdTimeFrame query
        const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
        // Determine updatedTimeFrame query
        const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
        // Find requested search array
        const searchResults = await (prisma[model] as BaseType).findMany({
            where: {
                ...where,
                ...idQuery,
                ...searchQuery,
                ...createdQuery,
                ...updatedQuery,
            },
            orderBy: sortQuery,
            take: input.take ?? 20,
            skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
            cursor: input.after ? {
                id: input.after
            } : undefined,
            ...select
        });
        // If there are results
        if (searchResults.length > 0) {
            // Find cursor
            const cursor = searchResults[searchResults.length - 1].id;
            // Query after the cursor to check if there are more results
            const hasNextPage = await (prisma[model] as BaseType).findMany({
                take: 1,
                cursor: {
                    id: cursor
                }
            });
            // Return results
            return {
                pageInfo: {
                    hasNextPage: hasNextPage.length > 0,
                    endCursor: cursor,
                },
                edges: searchResults.map((result: any) => ({
                    cursor: result.id,
                    node: toGraphQL(result),
                }))
            }
        }
        // If there are no results
        else {
            return {
                pageInfo: {
                    endCursor: null,
                    hasNextPage: false,
                },
                edges: []
            }
        }
    }
})

/**
 * Converts time frame to Prisma "where" query
 * @param time Time frame to convert
 * @param fieldName Name of time field (typically created_at or updated_at)
 */
export const timeFrameToPrisma = (fieldName: string, time?: TimeFrame | null | undefined): any => {
    if (!time || (!time.before && !time.after)) return undefined;
    let where: any = ({ [fieldName]: {} });
    if (time.before) where[fieldName].lte = time.before;
    if (time.after) where[fieldName].gte = time.after;
    return where;
}

/**
 * Compositional component for models which can be counted (e.g. for metrics)
 * @param state 
 * @returns 
 */
export const counter = <CountInput extends CountInputBase>(model: keyof PrismaModels, prisma?: PrismaType) => ({
    /**
     * Counts the number of objects in the database, optionally filtered by a where clauses
     * @param where Additional where clauses, in addition to the createdMetric and updatedMetric passed into input
     * @param input Count metrics common to all models
     * @returns The number of matching objects
     */
    async count(where: { [x: string]: any }, input: CountInput): Promise<number> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Create query for created metric
        const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
        // Create query for created metric
        const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
        // Count objects that match queries
        return await (prisma[model] as BaseType).count({
            where: {
                ...where,
                ...createdQuery,
                ...updatedQuery,
            },
        });
    }
})