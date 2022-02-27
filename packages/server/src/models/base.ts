// Components for providing basic functionality to model objects
import { Count, FindByIdInput, PageInfo, Success, TimeFrame } from '../schema/types';
import { PrismaType, RecursivePartial } from '../types';
import { GraphQLResolveInfo } from 'graphql';
import pkg from 'lodash';
import _ from 'lodash';
import { commentFormatter, commentSearcher } from './comment';
import { nodeFormatter } from './node';
import { organizationFormatter, organizationSearcher } from './organization';
import { projectFormatter, projectSearcher } from './project';
import { reportFormatter, reportSearcher } from './report';
import { resourceFormatter, resourceSearcher } from './resource';
import { roleFormatter } from './role';
import { routineFormatter, routineSearcher } from './routine';
import { standardFormatter, standardSearcher } from './standard';
import { tagFormatter, tagSearcher } from './tag';
import { userFormatter, userSearcher } from './user';
import { starFormatter } from './star';
import { voteFormatter } from './vote';
import { emailFormatter } from './email';
import { CustomError } from '../error';
import { CODE } from '@local/shared';
import { profileFormatter } from './profile';
import { memberFormatter } from './member';
import { resolveGraphQLInfo } from '../utils';
const { isObject } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export type FormatConverter<GraphQLModel> = {
    /**
     * Removes fields which are not in the database
     */
    removeCalculatedFields?: (partial: { [x: string]: any }) => any;
    /**
     * Convert database fields to GraphQL union types
     */
    constructUnions?: (data: { [x: string]: any }) => any;
    /**
     * Convert GraphQL unions to database fields
     */
    deconstructUnions?: (partial: { [x: string]: any }) => any;
    /**
     * Add join tables which are not present in GraphQL object
     */
    addJoinTables?: (partial: { [x: string]: any }) => any;
    /**
     * Remove join tables which are not present in GraphQL object
     */
    removeJoinTables?: (data: { [x: string]: any }) => any;
    /**
     * Adds fields which are calculated after the main query
     * @param userId ID of the user making the request
     * @param objects
     * @param info GraphQL info object or partial info object
     * @returns objects ready to be sent through GraphQL
     */
    addSupplementalFields?: (
        prisma: PrismaType,
        userId: string | null,
        objects: RecursivePartial<any>[],
        info: InfoType,
    ) => Promise<RecursivePartial<GraphQLModel>[]>;
}

/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<SearchInput> = {
    defaultSort: any;
    getSortQuery: (sortBy: string) => any;
    getSearchStringQuery: (searchString: string) => any;
    customQueries?: (input: SearchInput) => { [x: string]: any };
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

export interface ValidateMutationsInput<C, U> {
    userId: string | null,
    createMany?: C[] | null | undefined,
    updateMany?: U[] | null | undefined,
    deleteMany?: string[] | null | undefined,
}

export interface CUDInput<Create, Update> {
    userId: string | null,
    createMany?: Create[] | null | undefined,
    updateMany?: Update[] | null | undefined,
    deleteMany?: string[] | null | undefined,
    info: InfoType,
}

export interface CUDResult<GraphQLObject> {
    created?: RecursivePartial<GraphQLObject>[],
    updated?: RecursivePartial<GraphQLObject>[],
    deleted?: Count, // Number of deleted organizations
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
export const addJoinTablesHelper = (obj: any, map: JoinMap | undefined): any => {
    if (!map) return obj;
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object
        if (obj[key]) {
            // If the value is an array
            if (Array.isArray(obj[key])) {
                // Add the join table to each item in the array
                result[key] = obj[key].map((item: any) => ({ [value]: item }));
            } else {
                // Otherwise, add the join table to the object
                result[key] = { [value]: obj[key] };
            }
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
export const removeJoinTablesHelper = (obj: any, map: JoinMap | undefined): any => {
    if (!obj || !map) return obj;
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object
        if (obj[key]) {
            // If the value is an array
            if (Array.isArray(obj[key])) {
                // Remove the join table from each item in the array
                result[key] = obj[key].map((item: any) => item[value]);
            } else {
                // Otherwise, remove the join table from the object
                result[key] = obj[key][value];
            }
        }
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
export const addCountQueries = (obj: any, map: CountMap | undefined): any => {
    if (!map) return obj;
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
export const removeCreatorField = (select: any): any => {
    let { creator, ...rest } = select;
    return {
        ...rest,
        createdByUser: {
            __typename: 'User',
            id: true,
            username: true,
        },
        createdByOrganization: {
            __typename: 'Organization',
            id: true,
            name: true,
        }
    }
}

/**
 * Helper function for Prisma createdByUser/createdByOrganization fields to GraphQL creator field
 */
export const addCreatorField = (data: any): any => {
    let { createdByUser, createdByOrganization, ...rest } = data;
    return {
        ...rest,
        creator:
            createdByUser?.id
                ? createdByUser
                : createdByOrganization?.id
                    ? createdByOrganization
                    : null
    }
}

/**
 * Helper function for converting owner GraphQL field to Prisma user/organization fields
 */
export const removeOwnerField = (select: any): any => {
    let { creator, ...rest } = select;
    return {
        ...rest,
        user: {
            __typename: 'User',
            id: true,
            username: true,
        },
        organization: {
            __typename: 'Organization',
            id: true,
            name: true,
        }
    }
}

/**
 * Helper function for Prisma user/organization fields to GraphQL owner field
 */
export const addOwnerField = (data: any): any => {
    let { user, organization, ...rest } = data;
    return {
        ...rest,
        owner:
            user?.id
                ? user
                : organization?.id
                    ? organization
                    : null
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
 * Converts the {} values of a graphqlFields object to true. 
 * Skips formatting __typename
 */
export const formatGraphQLFields = (fields: { [x: string]: any }): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    Object.keys(fields).forEach((key) => {
        if (key === '__typename') converted[key] = fields[key];
        else if (Object.keys(fields[key]).length === 0) converted[key] = true;
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
 * NOTE: This function is idempotent. 
 * Converts a GraphQL info object to a partial Prisma select. 
 * This is then passed into a model-specific converter to handle virtual/calculated fields,
 * unions, and other special cases.
 * @param info - GraphQL info object, or result of this function
 * @param typemapper - Object that contains __typename of returning object 
 * and keys that map to typemappers for each possible relationship
 */
export const infoToPartialSelect = (info: InfoType, typeMapper): any => {
    //TODO temp
    console.log('WOOHOO NEW METHOD', resolveGraphQLInfo(info as GraphQLResolveInfo));
    // Return undefined if info not set
    if (!info) return undefined;
    // Find select fields in info object
    let select = info.hasOwnProperty('fieldName') ?
        resolveGraphQLInfo(info as GraphQLResolveInfo) :
        info;
    // The actual select we want is inside the "select" field
    select = select[Object.keys(select)[0]];
    // If fields are in the shape of a paginated search query, convert to a Prisma select object
    if (select.hasOwnProperty('pageInfo') && select.hasOwnProperty('edges')) {
        select = select.edges.node;
    }
    // Inject __typename fields
    //TODO
    return select;
}

/**
 * Helper function for creating a Prisma select object. 
 * If the select object is in the shape of a paginated search query, 
 * then it will be converted to a prisma select object.
 * @returns select object for Prisma operations
 */
export const selectHelper = (info: InfoType): any => {
    console.log('in selecthelper start', info)
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let partial = selectToDB(info);
    console.log('in selecthelper partial, info', partial, info)
    if (!_.isObject(partial)) return undefined;
    // Delete __typename fields
    partial = removeTypenames(partial);
    console.log('in selecthelper partial2, info', partial, info)
    // Pad every relationship with "select"
    const padded = padSelect(partial);
    console.log('in selecthelper padded, info', padded, info)
    return padded;
}

export const FormatterMap: { [x: string]: FormatConverter<any> } = {
    'Comment': commentFormatter(),
    'Email': emailFormatter(),
    'Member': memberFormatter(),
    'Node': nodeFormatter(),
    'Organization': organizationFormatter(),
    'Profile': profileFormatter(),
    'Project': projectFormatter(),
    'Report': reportFormatter(),
    'Resource': resourceFormatter(),
    'Role': roleFormatter(),
    'Routine': routineFormatter(),
    'Standard': standardFormatter(),
    'Star': starFormatter(),
    'Tag': tagFormatter(),
    'User': userFormatter(),
    'Vote': voteFormatter(),
}

export const SearcherMap: { [x: string]: Searcher<any> } = {
    'Comment': commentSearcher(),
    // 'Member': memberSearcher(),TODO create searchers for all these
    'Organization': organizationSearcher(),
    'Project': projectSearcher(),
    'Report': reportSearcher(),
    'Resource': resourceSearcher(),
    'Routine': routineSearcher(),
    'Standard': standardSearcher(),
    // 'Star': starSearcher(),
    'Tag': tagSearcher(),
    'User': userSearcher(),
    // 'Vote': voteSearcher(),
}

export const PrismaMap: { [x: string]: (prisma: PrismaType) => any } = {
    'Comment': (prisma: PrismaType) => prisma.comment,
    'Email': (prisma: PrismaType) => prisma.email,
    'Member': (prisma: PrismaType) => prisma.organization_users,
    'Node': (prisma: PrismaType) => prisma.node,
    'Organization': (prisma: PrismaType) => prisma.organization,
    'Profile': (prisma: PrismaType) => prisma.user,
    'Project': (prisma: PrismaType) => prisma.project,
    'Report': (prisma: PrismaType) => prisma.report,
    'Resource': (prisma: PrismaType) => prisma.resource,
    'Role': (prisma: PrismaType) => prisma.role,
    'Routine': (prisma: PrismaType) => prisma.routine,
    'Standard': (prisma: PrismaType) => prisma.standard,
    'Star': (prisma: PrismaType) => prisma.star,
    'Tag': (prisma: PrismaType) => prisma.tag,
    'User': (prisma: PrismaType) => prisma.user,
    'Vote': (prisma: PrismaType) => prisma.vote,
}

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

export enum RelationshipTypes {
    connect = 'connect',
    disconnect = 'disconnect',
    create = 'create',
    update = 'update',
    delete = 'delete',
}

export interface RelationshipToPrismaArgs {
    data: { [x: string]: any },
    relationshipName: string,
    isAdd: boolean,
    fieldExcludes?: string[],
    relExcludes?: RelationshipTypes[],
    softDelete?: boolean,
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
 * @param fieldExcludes Fields to exclude from the conversion
 * @param relExcludes Relationship types to exclude from the conversion
 * @param softDelete True if deletes should be converted to soft deletes
 */
export const relationshipToPrisma = ({
    data,
    relationshipName,
    isAdd,
    fieldExcludes = [],
    relExcludes = [],
    softDelete = false
}: RelationshipToPrismaArgs): {
    connect?: { [x: string]: any }[],
    disconnect?: { [x: string]: any }[],
    delete?: { [x: string]: any }[],
    create?: { [x: string]: any }[],
    update?: { [x: string]: any }[],
} => {
    // Determine valid operations, and remove operations that should be excluded
    let ops = isAdd ? [RelationshipTypes.connect, RelationshipTypes.create] : Object.values(RelationshipTypes);
    ops = _.difference(ops, relExcludes)
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    for (const [key, value] of Object.entries(data)) {
        // Skip if not matching relationship or not a valid operation
        if (!key.startsWith(relationshipName) || !ops.some(o => key.toLowerCase().endsWith(o))) continue;
        // Determine operation
        const currOp = key.replace(relationshipName, '').toLowerCase();
        // TODO handle soft delete
        // Add operation to result object
        const shapedData = shapeRelationshipData(value, fieldExcludes);
        converted[currOp] = Array.isArray(converted[currOp]) ? [...converted[currOp], ...shapedData] : shapedData;
    };
    return converted;
}

export interface JoinRelationshipToPrismaArgs extends RelationshipToPrismaArgs {
    joinFieldName: string
}

/**
 * Converts the result of relationshipToPrisma to apply to a many-to-many relationship 
 * (i.e. uses a join table).
 * @param data The data to convert
 * @param relationshipName The name of the relationship to convert (since data may contain irrelevant fields)
 * @param joinFieldName The name of the field in the join table associated with the child object
 * @param isAdd True if data is being converted for an add operation. This limits the prisma operations to only "connect" and "create"
 * @param fieldExcludes Fields to exclude from the conversion
 * @param relExcludes Relationship types to exclude from the conversion
 * @param softDelete True if deletes should be converted to soft deletes
 */
export const joinRelationshipToPrisma = ({
    data,
    relationshipName,
    joinFieldName,
    isAdd,
    fieldExcludes = [],
    relExcludes = [],
    softDelete = false
}: JoinRelationshipToPrismaArgs): {
    disconnect?: { [x: string]: any }[],
    delete?: { [x: string]: any }[],
    create?: { [x: string]: any }[],
    update?: { [x: string]: any }[],
} => {
    let converted: { [x: string]: any } = {};
    // Call relationshipToPrisma to get join data used for one-to-many relationships
    const normalJoinData = relationshipToPrisma({ data, relationshipName, isAdd, fieldExcludes, relExcludes, softDelete })
    // Convert this to support a join table
    if (normalJoinData.hasOwnProperty('connect')) {
        // ex: create: [{ tag: { connect: { id: '123' } } }]
        // Note that the connect is technically a create, since the join table row is created
        const dataArray = normalJoinData.connect?.map(e => ({
            [joinFieldName]: { connect: e }
        })) ?? [];
        converted.create = Array.isArray(converted.create) ? [...converted.create, ...dataArray] : dataArray;
    }
    if (normalJoinData.hasOwnProperty('disconnect')) {
        // ex: delete: [{ tag: { id: '123' } }]
        const dataArray = normalJoinData.disconnect?.map(e => ({
            [joinFieldName]: { e }
        })) ?? [];
        converted.delete = Array.isArray(converted.delete) ? [...converted.delete, ...dataArray] : dataArray;
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
        converted.create = Array.isArray(converted.create) ? [...converted.create, ...dataArray] : dataArray;
    }
    if (normalJoinData.hasOwnProperty('update')) {
        // ex: update: [{ tag: update: { id: '123' } } }]
        const dataArray = normalJoinData.create?.map(e => ({
            [joinFieldName]: { update: e }
        })) ?? [];
        converted.update = Array.isArray(converted.update) ? [...converted.update, ...dataArray] : dataArray;
    }
    return converted;
}

/**
 * Recursively removes fields which are not present in both objects
 * @param a First object
 * @param b Second object
 * @returns Recursive intersection of a and b
 */
export function omitDeep(a: { [x: string]: any }, b: { [x: string]: any }): { [x: string]: any } {
    const keys = Object.keys(a);
    const result: { [x: string]: any } = {};
    for (const key of keys) {
        if (b[key] !== undefined) {
            if (typeof a[key] === 'object') {
                result[key] = omitDeep(a[key], b[key]);
            } else {
                result[key] = a[key];
            }
        }
    }
    return result;
}

/**
 * Converts a GraphQL union into a Prisma select.
 * All possible union fields which the user wants to select are in the info partial, but they are not separated by type.
 * This function performs a nested intersection between each union type's avaiable fields, and the fields which were 
 * requested by the user.
 * @param partial - partial select object
 * @param unionField - Name of the union field
 * @param relationshipTuples - [name of Prisma relationship field, fields which should be selected if they are in partial]
 * @returns partial select object with GraphQL union converted into Prisma relationship selects
 */
export const deconstructUnion = (partial: any, unionField: string, relationshipTuples: [string, { [x: string]: any }][]): any => {
    let { [unionField]: unionData, ...rest } = partial;
    // If field in partial is not an object, return partial unmodified
    if (!unionData) return partial;
    for (const [relationshipName, selectFields] of relationshipTuples) {
        rest[relationshipName] = omitDeep(selectFields, partial);
    }
    return rest;
}

/**
 * Custom merge for object type dictionary arrays (e.g. { 'User': [{ id: '321' }], 'Standard': [{ id: '123' }, { id: '555'}] }). 
 * If an object within a type array appears in both a and b, its fields are merged.
 * @param a First object 
 * @param b Second object
 * @returns a and b merged together
 */
const mergeObjectTypeDict = (
    a: { [x: string]: { [x: string]: any }[] },
    b: { [x: string]: { [x: string]: any }[] }
): { [x: string]: { [x: string]: any }[] } => {
    const result: { [x: string]: { [x: string]: any }[] } = {};
    console.log('start of mergeObjectTypeDict', a, b);
    // Initialize result with a. If any of a's values are not arrays, make them arrays
    for (const [key, value] of Object.entries(a)) {
        if (Array.isArray(value)) {
            result[key] = value;
        } else {
            result[key] = [value];
        }
    }
    console.log('result after initializing', result);
    // Loop through b
    for (const [key, value] of Object.entries(b)) {
        // If key is not in result, add it
        if (!result[key]) {
            result[key] = value;
        }
        // If key is in result, merge values
        else {
            // Check if item in b array is already in result array for this key. If so, merge its values
            for (const item of value) {
                const index = result[key].findIndex(e => e.id === item.id);
                if (index !== -1) {
                    result[key][index] = { ...result[key][index], ...item };
                } else {
                    result[key].push(item);
                }
            }
        }
    }
    // for (const key of Object.keys(b)) {
    //     console.log('curr key', key);
    //     // If key shows up in both a and b, merge the arrays
    //     if (result[key]) {
    //         console.log(`key in both. Merging ${key}`);
    //         const curr = Array.isArray(b[key]) ? b[key] : [b[key]];
    //         console.log('curr', curr);
    //         // Loop through b array
    //         for (const item of curr) {
    //             console.log('in loopqwer', item);
    //             // Check if item in b array is already in result array for this key. If so, merge its values
    //             const index = result[key].findIndex(e => e.id === item.id);
    //             if (index !== -1) {
    //                 result[key][index] = { ...result[key][index], ...item };
    //             }
    //             // Otherwise, add it to the array
    //             else {
    //                 result[key].push(item);
    //             }
    //         }
    //         result[key] = a[key].concat(b[key]);
    //     }
    //     // Otherwise, just add it to the result 
    //     else {
    //         console.log('result[key] does not exist. just adding without merge', key);
    //         result[key] = Array.isArray(b[key]) ? [...b[key]] : [b[key]];
    //     }
    // }
    return result;
}

/**
 * Combines supplemental fields from a Prisma object with arbitrarily nested relationships
 * @param data GraphQL-shaped data, where each object contains at least an ID
 * @param info GraphQL info object, INCLUDING typenames (which is used to determine each object type)
 * @returns [objectDict, selectFieldsDict], where objectDict is a dictionary of object arrays (sorted by type), 
 * and selectFieldsDict is a dictionary of select fields for each type (unioned if they show up in multiple places)
 */
const groupSupplementsByType = (data: { [x: string]: any }, info: InfoType): [{ [x: string]: object[] }, { [x: string]: any }] => {
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    const partial = infoToPartialSelect(info);
    // Skip if __typename not in partial
    if (!partial.hasOwnProperty('__typename')) return [{}, {}];
    // Objects sorted by type. For now, objects keep their primitive fields to avoid recursion.
    // If in the future supplemental field requires nested data, this could be changed to call a 
    // custom formatter for each type.
    let objectDict: { [x: string]: { [x: string]: any }[] } = {};
    // Select fields for each type. If a type appears twice, the fields are combined.
    let selectFieldsDict: { [x: string]: { [x: string]: { [x: string]: any } } } = {};
    // Store primitives in a separate object
    const dataPrimitives: any = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // TODO temp
            if (key === 'tags') console.log('groupsupplementsbytype isarray tags', partial[key]);
            // If __typename for key does not exist (i.e. it was not requested), skip
            if (!partial.hasOwnProperty(key) || !partial[key]?.__typename) continue;
            if (key === 'tags') console.log('made it past check');
            selectFieldsDict[partial[key].__typename] = _.merge(selectFieldsDict[partial[key].__typename] ?? {}, partial[key]);
            // Pass each element through groupSupplementsByType
            for (const v of value) {
                if (key === 'tags') console.log('passing element through groupSupplementsByType', v);
                const [childObjectsDict, childSelectFieldsDict] = groupSupplementsByType(v, partial[key]);
                if (key === 'tags') console.log('passed element through groupsupplementstype', childObjectsDict);
                if (key === 'tags') console.log('objectdict before child merge:', objectDict);
                objectDict = mergeObjectTypeDict(objectDict, childObjectsDict);
                if (key === 'tags') console.log('merged objectdict!', objectDict);
                selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
            }
        }
        // If value is an object (and not date), add it to the correct key in objectDict
        else if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            // If __typename for key does not exist (i.e. it was not requested), skip
            if (!partial.hasOwnProperty(key) || !partial[key]?.__typename) continue;
            selectFieldsDict[partial[key].__typename] = _.merge(selectFieldsDict[partial[key].__typename] ?? {}, partial[key]);
            // Pass value through groupSupplementsByType
            const [childObjectsDict, childSelectFieldsDict] = groupSupplementsByType(value, partial[key]);
            objectDict = mergeObjectTypeDict(objectDict, childObjectsDict);
            selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
        }
        // Otherwise, add it to the primitives array
        else {
            dataPrimitives[key] = value;
        }
    }
    // Handle base case
    // Remove anything that's not a primitive from partial and data
    const partialPrimitives = Object.keys(partial)
        .filter(key => !Array.isArray(partial[key]) && !partial[key]?.__typename)
        .reduce((res: any, key) => (res[key] = partial[key], res), {});
    // Finally, add the base object to objectDict (primitives only) and selectFieldsDict
    if (partial.__typename in objectDict) {
        objectDict[partial.__typename].push(dataPrimitives);
    } else {
        objectDict[partial.__typename] = [dataPrimitives];
    }
    selectFieldsDict[partial.__typename] = _.merge(selectFieldsDict[partial.__typename] ?? {}, partialPrimitives);
    // Return objectDict and selectFieldsDict
    return [objectDict, selectFieldsDict];
}

/**
 * Recombines objects from supplementalFields calls into shape that matches info
 * @param data Original, unsupplemented data, where each object has an ID
 * @param objectsById Dictionary of objects with supplemental fields added, where each value contains at least an ID
 * @returns data with supplemental fields added
 */
const combineSupplements = (data: { [x: string]: any }, objectsById: { [x: string]: any }) => {
    console.log('in combinesupplements start')
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through combineSupplements
            data[key] = data[key].map((v: any) => combineSupplements(v, objectsById));
        }
        // If value is an object (and not a date), add it to the correct key in objectDict
        else if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            // Pass value through combineSupplements
            data[key] = combineSupplements(value, objectsById);
        }
    }
    console.log('combinesupplements base case ;)', _.merge(data, objectsById[data.id]));
    // Handle base case
    return _.merge(data, objectsById[data.id])
}

/**
 * Adds supplemental fields to the select object, and all of its relationships (and their relationships, etc.)
 * Groups objects types together, so database is called only once for each type.
 * @param prisma Prisma client
 * @param userId Requesting user's ID
 * @param data Array of GraphQL-shaped data, where each object contains at least an ID
 * @param info GraphQL info object, INCLUDING typenames (which is used to determine each object type)
 * @returns data array with supplemental fields added to each object
 */
export const addSupplementalFields = async (prisma: PrismaType, userId: string | null, data: { [x: string]: any }[], info: InfoType): Promise<{ [x: string]: any }[]> => {
    console.log('addsupplementalfields start', (info as any)?.__typename);
    // Group objects by type
    let objectDict: { [x: string]: object[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: any } } = {};
    for (const d of data) {
        //TODO temp. check all tags which should be in objectdict.currently only 'Cardano' shows up
        if (d.tags && d.tags.length > 0) console.log('addsupplementalfields d.tags start', d.tags);
        const [childObjectsDict, childSelectFieldsDict] = groupSupplementsByType(d, info);
        // Merge each array in childObjectsDict into objectDict
        objectDict = mergeObjectTypeDict(objectDict, childObjectsDict);//TODO temp. something wrong with merging. make sure all tags return correctly in one array
        if (d.tags && d.tags.length > 0) console.log('addsupplementalfields d.tags objectsdict merged', objectDict);
        // Merge each object in childSelectFieldsDict into selectFieldsDict
        selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
    }
    console.log('objectDict', (info as any)?.__typename, objectDict); //TODO check here if all tags are there
    // Dictionary to store objects by ID
    const objectsById: { [x: string]: any } = {};
    // Pass each objectDict value through the correct custom supplemental field function
    for (const [key, value] of Object.entries(objectDict)) {
        console.log('keeeeeeeeeeeey', key)
        if (key in FormatterMap) {
            const valuesWithSupplements = FormatterMap[key]?.addSupplementalFields
                ? await (FormatterMap[key] as any).addSupplementalFields(prisma, userId, value, selectFieldsDict[key])
                : value;
            console.log('called object addSupplementalFields', key, valuesWithSupplements);
            // Add each value to objectsById
            for (const v of valuesWithSupplements) {
                console.log('adding to objectsById', v.id);
                objectsById[v.id] = v;
            }
        }
    }
    console.log('going to recursive combineSupplements', objectsById); //TODO only 9ea8 Cardano tag shows up here
    let result = data.map(d => combineSupplements(d, objectsById));
    console.log('got result', (info as any)?.__typename, result);
    //TODO temp
    if (result?.length > 0 && result[0].tags) console.log('got result tags', result[0].tags); // TODO isStarred missing on FIRST tag in result
    // Convert objectsById dictionary back into shape of data
    return result
}

/**
 * Shapes GraphQL info object to become a valid Prisma select
 * @param info GraphQL info object, INCLUDING typenames (which is used to determine each object type)
 * @returns Valid Prisma select object
 */
export const selectToDB = (info: InfoType): { [x: string]: any } => {
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    // Also perform clone to make sure that info is not modified
    let partial = infoToPartialSelect(JSON.parse(JSON.stringify(info)));
    // Loop through each key/value pair in info
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recursively call selectToDB
        if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            partial[key] = selectToDB(value);
        }
    }
    // Handle base case
    if (partial.__typename in FormatterMap) {
        const formatter = FormatterMap[partial.__typename];
        // Remove calculated fields and/or add fields for calculating
        if (formatter.removeCalculatedFields) partial = formatter.removeCalculatedFields(partial);
        // Deconstruct unions
        if (formatter.deconstructUnions) partial = formatter.deconstructUnions(partial);
        // Add join tables
        if (formatter.addJoinTables) partial = formatter.addJoinTables(partial);
    }
    return partial;
}

/**
 * Shapes Prisma model object to become a valid GraphQL model object
 * @param data Prisma object
 * @param info GraphQL info object, INCLUDING typenames (which is used to determine each object type)
 * @returns Valid GraphQL object
 */
export const modelToGraphQL = (data: { [x: string]: any }, info: InfoType): { [x: string]: any } => {
    console.log('in modelToGraphQL', (info as any)?.__typename, data);
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = infoToPartialSelect(info);
    console.log('partial in modeltographql', partial);
    // First convert data to usable shape
    if (partial?.__typename in FormatterMap) {
        const formatter = FormatterMap[partial.__typename];
        // Construct unions
        if (formatter.constructUnions) data = formatter.constructUnions(data);
        console.log('constructed union', data)
        // Remove join tables
        if (formatter.removeJoinTables) data = formatter.removeJoinTables(data);
        console.log('removed tables', data)
    }
    // Then loop through each key/value pair in data and call modelToGraphQL on each array item/object
    for (const [key, value] of Object.entries(data)) {
        console.log('modeltographql loop', key, partial[key], value);
        // If key doesn't exist in partial, skip
        if (!_.isObject(partial) || !(partial as any)[key]) continue;
        // If value is an array, call modelToGraphQL on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGraphQL
            data[key] = data[key].map((v: any) => modelToGraphQL(v, partial[key]));
        }
        // If value is an object (and not date), call modelToGraphQL on it
        else if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            data[key] = modelToGraphQL(value, (partial as any)[key]);
        }
    }
    return data;
}

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @param userId ID of user making the request
 * @param input GraphQL create input object
 * @param info GraphQL info object
 * @param cud Object's CUD helper function
 * @param prisma Prisma client
 * @returns GraphQL response object
 */
export async function createHelper<CreateInput, UpdateInput, GraphQLOutput>(
    userId: string | null,
    input: any,
    info: InfoType,
    cud: (input: CUDInput<CreateInput, UpdateInput>) => Promise<CUDResult<GraphQLOutput>>,
    prisma: PrismaType
): Promise<RecursivePartial<GraphQLOutput>> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object');
    const { created } = await cud({ info, userId, createMany: [input] });
    if (created && created.length > 0) {
        return await addSupplementalFields(prisma, userId, created, info) as any;
    }
    throw new CustomError(CODE.ErrorUnknown);
}

/**
 * Helper function for reading one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL find by ID input object
 * @param info GraphQL info object
 * @param prisma Prisma object
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel>(
    userId: string | null,
    input: FindByIdInput,
    info: InfoType,
    prisma: PrismaType
): Promise<RecursivePartial<GraphQLModel>> {
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = infoToPartialSelect(info);
    // Uses __typename to determine which Prisma object is being queried
    const objectType = partial.__typename;
    if (!(objectType in PrismaMap)) {
        console.log('uh oh spaghetti o\'', partial);
        throw new CustomError(CODE.InternalError, `${objectType} not found`);
    }
    // Get the Prisma object
    let object = await PrismaMap[objectType](prisma).findUnique(prisma, { where: { id: input.id }, ...selectHelper(info) });
    if (!object) throw new CustomError(CODE.NotFound, `${objectType} not found`);
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, info);
    return (await addSupplementalFields(prisma, userId, [formatted], info))[0] as RecursivePartial<GraphQLModel>;
}

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * @param userId ID of user making the request
 * @param input GraphQL search input object
 * @param info GraphQL info object
 * @param prisma Prisma object
 * @param searcher Searcher object
 * @param additionalQueries Additional where clauses to apply to the search
 * @returns Paginated search result
 */
export async function readManyHelper<SearchInput extends SearchInputBase<any>>(
    userId: string | null,
    input: SearchInput,
    info: InfoType,
    prisma: PrismaType,
    searcher: Searcher<SearchInput>,
    additionalQueries?: { [x: string]: any },
): Promise<PaginatedSearchResult> {
    console.log('readmanyhelper start', info)
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = infoToPartialSelect(info);
    console.log('readmanyhelper partial info', partial, info)
    // Uses __typename to determine which Prisma object is being queried
    const objectType = partial.__typename;
    if (!(objectType in PrismaMap)) throw new CustomError(CODE.InternalError, `${objectType} not found`);
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
    // Determine text search query
    const searchQuery = input.searchString ? searcher.getSearchStringQuery(input.searchString) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    const typeQuery = SearcherMap[objectType]?.customQueries ? (SearcherMap[objectType] as any).customQueries(input) : undefined;
    // Combine queries
    const where = { ...additionalQueries, ...idQuery, ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
    // Determine sort order
    const orderBy = searcher.getSortQuery(input.sortBy ?? searcher.defaultSort);
    // Find requested search array
    const searchResults = await PrismaMap[objectType](prisma).findMany({
        where,
        orderBy,
        take: input.take ?? 20,
        skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        cursor: input.after ? {
            id: input.after
        } : undefined,
        ...selectHelper(info)
    });
    // If there are results
    let paginatedResults: PaginatedSearchResult;
    if (searchResults.length > 0) {
        // Find cursor
        const cursor = searchResults[searchResults.length - 1].id;
        // Query after the cursor to check if there are more results
        const hasNextPage = await PrismaMap[objectType](prisma).findMany({
            take: 1,
            cursor: {
                id: cursor
            }
        });
        paginatedResults = {
            pageInfo: {
                hasNextPage: hasNextPage.length > 0,
                endCursor: cursor,
            },
            edges: searchResults.map((result: any) => ({
                cursor: result.id,
                node: result,
            }))
        }
    }
    // If there are no results
    else {
        paginatedResults = {
            pageInfo: {
                endCursor: null,
                hasNextPage: false,
            },
            edges: []
        }
    }
    console.log('chicken bobicken partial info', partial, info)
    // Return formatted for GraphQL
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    console.log('findmany nodes before modeltographql', formattedNodes);
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, info));
    console.log('findmany nodes before addSupplementalFields', formattedNodes);
    formattedNodes = await addSupplementalFields(prisma, userId, formattedNodes, info);
    console.log('findmany nodes after addSupplementalFields', formattedNodes);
    //TODO temp
    if (formattedNodes.length >= 20) {
        console.log('20th item tag', formattedNodes[19]?.tags);
        if (formattedNodes[19]?.tags?.length > 0) {
            console.log('20th item FIRST TAG', formattedNodes[19]?.tags[0]);
        }
        if (formattedNodes[19]?.tags?.length > 1) {
            console.log('20th item SECOND TAG', formattedNodes[19]?.tags[1]);
        }
        if (formattedNodes[19]?.tags?.length > 2) {
            console.log('20th item THIRD TAG', formattedNodes[19]?.tags[2]);
        }
    }
    return { pageInfo: paginatedResults.pageInfo, edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
}

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @param input Count metrics common to all models
 * @param objectType __typename of object
 * @param prisma Prisma object
 * @param where Additional where clauses, in addition to the createdMetric and updatedMetric passed into input
 * @returns The number of matching objects
 */
export async function countHelper<CountInput extends CountInputBase>(input: CountInput, objectType: string, prisma: PrismaType, where?: { [x: string]: any }): Promise<number> {
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Count objects that match queries
    return await PrismaMap[objectType](prisma).count({
        where: {
            ...where,
            ...createdQuery,
            ...updatedQuery,
        },
    });
}

/**
 * Helper function for updating one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL update input object
 * @param info GraphQL info object
 * @param cud Object's CUD helper function
 * @param prisma Prisma client
 * @returns GraphQL response object
 */
export async function updateHelper<CreateInput, UpdateInput, GraphQLModel>(
    userId: string | null,
    input: any,
    info: InfoType,
    cud: (input: CUDInput<CreateInput, UpdateInput>) => Promise<CUDResult<GraphQLModel>>,
    prisma: PrismaType,
): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object');
    const { updated } = await cud({ info, userId, updateMany: [input] });
    if (updated && updated.length > 0) {
        return await addSupplementalFields(prisma, userId, updated, info) as any;
    }
    throw new CustomError(CODE.ErrorUnknown);
}

/**
 * Helper function for deleting one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param cud Object's CUD helper function
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper<CreateInput, UpdateInput, GraphQLModel>(
    userId: string | null,
    input: any,
    cud: (input: CUDInput<CreateInput, UpdateInput>) => Promise<CUDResult<GraphQLModel>>,
): Promise<Success> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete object');
    const { deleted } = await cud({ info: {}, userId, deleteMany: [input.id] });
    return { success: Boolean((deleted as any)?.count > 0) };
}

/**
 * Helper function for deleting many of the same object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param cud Object's CUD helper function
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper<CreateInput, UpdateInput, GraphQLModel>(
    userId: string | null,
    input: any,
    cud: (input: CUDInput<CreateInput, UpdateInput>) => Promise<CUDResult<GraphQLModel>>,
): Promise<Count> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete objects');
    const { deleted } = await cud({ info: {}, userId, deleteMany: [input.id] });
    if (!deleted) throw new CustomError(CODE.ErrorUnknown);
    return deleted
}