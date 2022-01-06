// Components for providing basic functionality to model objects
import { CODE } from '@local/shared';
import { PrismaSelect } from '@paljs/plugins';
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, PageInfo, InputMaybe, ReportInput, Scalars, MetricTimeFrame } from '../schema/types';
import { CustomError } from '../error';
import { PrismaType, RecursivePartial } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLObjectType, GraphQLResolveInfo } from 'graphql';
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
    joinMapper?: JoinMap;
    toDB: (obj: RecursivePartial<GraphQLModel>) => RecursivePartial<FullDBModel>;
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

type InfoType = GraphQLResolveInfo | { [x: string]: any } | null;

type PaginatedSearchResult = {
    pageInfo: PageInfo;
    edges: Array<{
        cursor: string;
        node: any;
    }>;
}

type SearchInputBase<SortBy> = {
    ids?: string[] | null;
    searchString?: string | null;
    sortBy?: SortBy | null;
    after?: string | null;
    take?: number | null;
}

type CountInputBase = {
    createdMetric?: MetricTimeFrame | null;
    updatedMetric?: MetricTimeFrame | null;
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
    console.log('INFO', info);
    console.log('a', info.hasOwnProperty('fieldName'));
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
        // Find requested search array
        const searchResults = await (prisma[model] as BaseType).findMany({
            where: {
                ...where,
                ...idQuery,
                ...searchQuery,
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
 * Compositional component for models which can be counted (e.g. for metrics)
 * @param state 
 * @returns 
 */
export const counter = <CountInput extends CountInputBase>(model: keyof PrismaModels, prisma?: PrismaType) => ({
    /**
     * Converts time frame enum to milliseconds
     * @param time Time frame to convert
     * @returns Time frame in milliseconds
     */
    timeConverter: (time: MetricTimeFrame) => {
        switch (time) {
            case MetricTimeFrame.Daily:
                return 86400000;
            case MetricTimeFrame.Weekly:
                return 604800000;
            case MetricTimeFrame.Monthly:
                return 2592000000;
            case MetricTimeFrame.Yearly:
                return 31536000000;
            default:
                return 86400000;
        }
    },
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
        const createdQuery = input.createdMetric ? ({ created_at: { gt: new Date(Date.now() - this.timeConverter(input.createdMetric)) } }) : undefined;
        // Create query for updated metric
        const updatedQuery = input.updatedMetric ? ({ updated_at: { gt: new Date(Date.now() - this.timeConverter(input.updatedMetric)) } }) : undefined;
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