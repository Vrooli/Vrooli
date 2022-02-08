// Components for providing basic functionality to model objects
import { PageInfo, TimeFrame } from '../schema/types';
import { PrismaType, RecursivePartial } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
import graphqlFields from 'graphql-fields';
import pkg from 'lodash';
const { isObject } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

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
     * @param obj GraphQL object to convert
     */
    toDB: (obj: RecursivePartial<GraphQLModel>) => RecursivePartial<FullDBModel>;
    /**
     * Converts object from Prisma representation to GraphQL
     * @param obj - Prisma object to convert
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

export type BaseType = PrismaModels['comment']; // It doesn't matter what PrismaType is used here, it's just to help TypeScript handle Prisma operations

// Strings for accessing model functions from Prisma
export const MODEL_TYPES = {
    Comment: 'comment',
    Email: 'email',
    Node: 'node',
    Organization: 'organization',
    Project: 'project',
    Report: 'report',
    Resource: 'resource',
    Role: 'role',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
    User: 'user',
    Vote: 'vote',
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
 * Helper function for converting creator GraphQL field to Prisma createdByUser/createdByOrganization fields
 */
export const removeCreatorField = (modified: any): any => {
    modified.createdByUser = {
        id: true,
        username: true,
    }
    modified.createdByOrganization = {
        id: true,
        name: true,
    }
    delete modified.creator;
    return modified;
}

/**
 * Helper function for Prisma createdByUser/createdByOrganization fields to GraphQL creator field
 */
export const addCreatorField = (modified: any): any => {
    if (modified.createdByUser?.id) {
        modified.creator = modified.createdByUser;
    } else if (modified.createdByOrganization?.id) {
        modified.creator = modified.createdByOrganization;
    }
    else modified.creator = null;
    delete modified.createdByUser;
    delete modified.createdByOrganization;
    return modified;
}

/**
 * Helper function for converting owner GraphQL field to Prisma user/organization fields
 */
export const removeOwnerField = (modified: any): any => {
    modified.user = {
        id: true,
        username: true,
    }
    modified.organization = {
        id: true,
        name: true,
    }
    delete modified.owner;
    return modified;
}

/**
 * Helper function for Prisma user/organization fields to GraphQL owner field
 */
export const addOwnerField = (modified: any): any => {
    if (modified.user?.id) {
        modified.owner = modified.user;
    } else if (modified.organization?.id) {
        modified.owner = modified.organization;
    }
    else modified.owner = null;
    delete modified.user;
    delete modified.organization;
    return modified;
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
export const formatGraphQLFields = (fields: { [x: string]: any }): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (Object.keys(fields[key]).length === 0) converted[key] = true;
        else converted[key] = formatGraphQLFields(fields[key]);
    });
    return converted;
}

/**
 * Removes the "__typename" field recursively from a JSON-serializable object
 * @param obj - JSON-serializable object with possible __typename fields
 * @return obj without __typename fields
 */
export const removeTypenames = (obj: { [x: string]: any }): { [x: string]: any } => {
    return JSON.parse(JSON.stringify(obj, (k, v) => (k === '__typename') ? undefined : v))
}


/**
 * Adds "select" to the correct parts of an object to make it a Prisma select
 */
export const padSelect = (fields: { [x: string]: any }): { [x: string]: any } => {
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
    // Make sure to delete all occurrences of the __typename field
    select = removeTypenames(select);
    return padSelect(select);
}

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
    prisma: PrismaType) => ({
        /**
         * Cursor-based search. Supports pagination, sorting, and filtering by string.
         * @param where Additional where clauses to apply to the search
         * @param input GraphQL-provided search parameters
         * @param info Requested return information
         */
        async search(where: { [x: string]: any }, input: SearchInput, info: InfoType): Promise<PaginatedSearchResult> {
            const boop = selectHelper<GraphQLModel, FullDBModel>(info, toDB);
            console.log('SEARCH BOIII', boop)
            console.log('SEARCH BOIII _COUNT', boop._count)
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
export const counter = <CountInput extends CountInputBase>(model: keyof PrismaModels, prisma: PrismaType) => ({
    /**
     * Counts the number of objects in the database, optionally filtered by a where clauses
     * @param where Additional where clauses, in addition to the createdMetric and updatedMetric passed into input
     * @param input Count metrics common to all models
     * @returns The number of matching objects
     */
    async count(where: { [x: string]: any }, input: CountInput): Promise<number> {
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

/**
 * Filters excluded fields from an object
 * @param data The object to filter
 * @param excludes The fields to exclude
 */
const filterFields = (data: { [x: string]: any }, excludes: string[]): { [x: string]: any } => {
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    Object.keys(data).forEach((key) => {
        // If key is not in excludes, add to result
        if (!excludes.some(e => e === key)) {
            converted[key] = data[key];
        }
    });
    return converted;
}

/**
 * Helper method to shape Prisma connect, disconnect, create, update, and delete data
 * Examples:
 *  - '123' => [{ id: '123' }]
 *  - { id: '123' } => [{ id: '123' }]
 *  - { name: 'John' } => [{ name: 'John' }]
 *  - ['123', '456'] => [{ id: '123' }, { id: '456' }]
 * @param data The data to shape
 * @param excludes The fields to exclude from the shape
 */
const shapeRelationshipData = (data: any, excludes: string[] = []): any => {
    if (Array.isArray(data)) {
        return data.map(e => {
            if (isObject(e)) {
                return filterFields(e, excludes);
            } else {
                return { id: e };
            }
        });
    } else if (isObject(data)) {
        return [filterFields(data, excludes)];
    } else {
        return [{ id: data }];
    }
}

/**
 * Converts an add or update's data to proper Prisma format. 
 * NOTE1: Must authenticate before calling this function!
 * NOTE2: Only goes one layer deep. You must handle grandchildren, great-grandchildren, etc. yourself
 * ex: { childConnect: [...], childCreate: [...], childDelete: [...] } => 
 *     { child: { connect: [...], create: [...], deleteMany: [...] } }
 * @param data The data to convert
 * @param relationshipName The name of the relationship to convert (since data may contain irrelevant fields)
 * @param isAdd True if data is being converted for an add operation. This limits the prisma operations to only "connect" and "create"
 * @param exclues Fields to exclude from the conversion
 * @param softDelete True if deletes should be converted to soft deletes
 */
export const relationshipToPrisma = (
    data: { [x: string]: any },
    relationshipName: string,
    isAdd: boolean,
    excludes: string[] = [],
    softDelete: boolean = false): {
        connect?: { [x: string]: any }[],
        disconnect?: { [x: string]: any }[],
        delete?: { [x: string]: any }[],
        create?: { [x: string]: any }[],
        update?: { [x: string]: any }[],
    } => {
    // Determine valid operations
    const ops = isAdd ? ['Connect', 'Create'] : ['Connect', 'Disconnect', 'Delete', 'Create', 'Update'];
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    for (const [key, value] of Object.entries(data)) {
        // Skip if not matching relationship or not a valid operation
        if (!key.startsWith(relationshipName) || !ops.some(o => key.endsWith(o))) continue;
        // Determine operation
        const currOp = key.replace(relationshipName, '');
        // TODO handle soft delete
        // Add operation to result object
        const shapedData = shapeRelationshipData(value, excludes);
        converted[currOp.toLowerCase()] = Array.isArray(converted[currOp.toLowerCase()]) ? [...converted[currOp.toLowerCase()], ...shapedData] : shapedData;
    };
    return converted;
}

/**
 * Converts the result of relationshipToPrisma to apply to a many-to-many relationship 
 * (i.e. uses a join table).
 * @param data The data to convert
 * @param relationshipName The name of the relationship to convert (since data may contain irrelevant fields)
 * @param joinFieldName The name of the field in the join table associated with the child object
 * @param isAdd True if data is being converted for an add operation. This limits the prisma operations to only "connect" and "create"
 * @param exclues Fields to exclude from the conversion
 * @param softDelete True if deletes should be converted to soft deletes
 */
export const joinRelationshipToPrisma = (
    data: { [x: string]: any },
    relationshipName: string,
    joinFieldName: string,
    isAdd: boolean,
    excludes: string[] = [],
    softDelete: boolean = false): {
        disconnect?: { [x: string]: any }[],
        delete?: { [x: string]: any }[],
        create?: { [x: string]: any }[],
        update?: { [x: string]: any }[],
    } => {
    console.log('in joinRelationshipToPrisma', relationshipName, joinFieldName);
    let converted: { [x: string]: any } = {};
    // Call relationshipToPrisma to get join data used for one-to-many relationships
    const normalJoinData = relationshipToPrisma(data, relationshipName, isAdd, excludes, softDelete);
    // Convert this to support a join table
    if (normalJoinData.hasOwnProperty('connect')) {
        console.log('in connect', normalJoinData.connect);
        // ex: create: [{ tag: { connect: { id: '123' } } }]
        // Note that the connect is technically a create, since the join table row is created
        const dataArray = normalJoinData.connect?.map(e => ({
            [joinFieldName]: { connect: e }
        })) ?? [];
        converted.create = Array.isArray(normalJoinData.create) ? [...normalJoinData.create, ...dataArray] : dataArray;
    }
    if (normalJoinData.hasOwnProperty('disconnect')) {
        // ex: delete: [{ tag: { id: '123' } }]
        const dataArray = normalJoinData.disconnect?.map(e => ({
            [joinFieldName]: { e }
        })) ?? [];
        converted.delete = Array.isArray(normalJoinData.delete) ? [...normalJoinData.delete, ...dataArray] : dataArray;
    }
    if (normalJoinData.hasOwnProperty('delete')) {
        // ex: tag: { delete: [{ id: '123' }, { id: '432' }] }
        converted[joinFieldName] = { delete: normalJoinData.delete };
    }
    if (normalJoinData.hasOwnProperty('create')) {
        // ex: create: [{ tag: { create: { id: '123' } } }]
        const dataArray = normalJoinData.create?.map(e => ({
            [joinFieldName]: { create: e }
        })) ?? [];
        converted.create = Array.isArray(normalJoinData.create) ? [...normalJoinData.create, ...dataArray] : dataArray;
    }
    if (normalJoinData.hasOwnProperty('update')) {
        // ex: update: [{ tag: update: { id: '123' } } }]
        const dataArray = normalJoinData.create?.map(e => ({
            [joinFieldName]: { update: e }
        })) ?? [];
        converted.update = Array.isArray(normalJoinData.update) ? [...normalJoinData.update, ...dataArray] : dataArray;
    }
    return converted;
}
