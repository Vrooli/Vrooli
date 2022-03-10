// Components for providing basic functionality to model objects
import { Count, FindByIdInput, PageInfo, Success, TimeFrame } from '../schema/types';
import { PrismaType, RecursivePartial } from '../types';
import { GraphQLResolveInfo } from 'graphql';
import pkg from 'lodash';
import _ from 'lodash';
import { commentFormatter, commentSearcher } from './comment';
import { nodeFormatter, nodeRoutineListFormatter } from './node';
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
import { inputItemFormatter } from './inputItem';
import { outputItemFormatter } from './outputItem';
import { resourceListFormatter, resourceListSearcher } from './resourceList';
import { tagHiddenFormatter } from './tagHidden';
const { isObject } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

export enum GraphQLModelType {
    Comment = 'Comment',
    Email = 'Email',
    InputItem = 'InputItem',
    Member = 'Member',
    Node = 'Node',
    NodeEnd = 'NodeEnd',
    NodeLoop = 'NodeLoop',
    NodeRoutineList = 'NodeRoutineList',
    Organization = 'Organization',
    OutputItem = 'OutputItem',
    Profile = 'Profile',
    Project = 'Project',
    Report = 'Report',
    Resource = 'Resource',
    ResourceList = 'ResourceList',
    Role = 'Role',
    Routine = 'Routine',
    Standard = 'Standard',
    Star = 'Star',
    Tag = 'Tag',
    TagHidden = 'TagHidden',
    User = 'User',
    Vote = 'Vote',
    Wallet = 'Wallet',
}

/**
 * Basic structure of an object's business layer.
 * Every business layer object has at least a PrismaType object and a format converter. 
 * Everything else is optional
 */
export interface ModelBusinessLayer<GraphQLModel, SearchInput> extends FormatConverter<GraphQLModel>, Partial<Searcher<SearchInput>> {
    prisma: PrismaType,
    cud?({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<any, any>): Promise<CUDResult<GraphQLModel>>
}

/**
 * Partial conversion between GraphQLResolveInfo and the select shape
 * used by Prisma's connect, disconnect, create, update, and delete methods.
 * Each level contains a __typename field
 */
export interface PartialInfo {
    [x: string]: GraphQLModelType | undefined | boolean | { [x: string]: PartialInfo };
    __typename?: GraphQLModelType;
}

export type RelationshipMap = { [relationshipName: string]: GraphQLModelType | { [fieldName: string]: GraphQLModelType } };

/**
 * Helper functions for converting between Prisma types and GraphQL types
 */
export type FormatConverter<GraphQLModel> = {
    /**
     * Maps relationship names to their GraphQL type. 
     * If the relationship is a union (i.e. has mutliple possible types), 
     * the GraphQL type will be an object of field/GraphQLModelType pairs.
     */
    relationshipMap: RelationshipMap;
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
        partial: PartialInfo,
    ) => Promise<RecursivePartial<GraphQLModel>[]>;
}

/**
 * Describes shape of component that can be sorted in a specific order
 */
export type Searcher<SearchInput> = {
    defaultSort: any;
    getSortQuery: (sortBy: string) => any;
    getSearchStringQuery: (searchString: string, languages?: string[]) => any;
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
    partial: PartialInfo,
}

export interface CUDResult<GraphQLObject> {
    created?: RecursivePartial<GraphQLObject>[],
    updated?: RecursivePartial<GraphQLObject>[],
    deleted?: Count, // Number of deleted organizations
}

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

export const FormatterMap: { [x: string]: FormatConverter<any> } = {
    [GraphQLModelType.Comment]: commentFormatter(),
    [GraphQLModelType.Email]: emailFormatter(),
    [GraphQLModelType.InputItem]: inputItemFormatter(),
    [GraphQLModelType.Member]: memberFormatter(),
    [GraphQLModelType.Node]: nodeFormatter(),
    [GraphQLModelType.NodeRoutineList]: nodeRoutineListFormatter(),
    [GraphQLModelType.Organization]: organizationFormatter(),
    [GraphQLModelType.OutputItem]: outputItemFormatter(),
    [GraphQLModelType.Profile]: profileFormatter(),
    [GraphQLModelType.Project]: projectFormatter(),
    [GraphQLModelType.Report]: reportFormatter(),
    [GraphQLModelType.Resource]: resourceFormatter(),
    [GraphQLModelType.ResourceList]: resourceListFormatter(),
    [GraphQLModelType.Role]: roleFormatter(),
    [GraphQLModelType.Routine]: routineFormatter(),
    [GraphQLModelType.Standard]: standardFormatter(),
    [GraphQLModelType.Star]: starFormatter(),
    [GraphQLModelType.Tag]: tagFormatter(),
    [GraphQLModelType.TagHidden]: tagHiddenFormatter(),
    [GraphQLModelType.User]: userFormatter(),
    [GraphQLModelType.Vote]: voteFormatter(),
}

export const SearcherMap: { [x: string]: Searcher<any> } = {
    [GraphQLModelType.Comment]: commentSearcher(),
    // 'Member': memberSearcher(),TODO create searchers for all these
    [GraphQLModelType.Organization]: organizationSearcher(),
    [GraphQLModelType.Project]: projectSearcher(),
    [GraphQLModelType.Report]: reportSearcher(),
    [GraphQLModelType.Resource]: resourceSearcher(),
    [GraphQLModelType.ResourceList]: resourceListSearcher(),
    [GraphQLModelType.Routine]: routineSearcher(),
    [GraphQLModelType.Standard]: standardSearcher(),
    // 'Star': starSearcher(),
    [GraphQLModelType.Tag]: tagSearcher(),
    [GraphQLModelType.User]: userSearcher(),
    // 'Vote': voteSearcher(),
}

export const PrismaMap: { [x: string]: (prisma: PrismaType) => any } = {
    [GraphQLModelType.Comment]: (prisma: PrismaType) => prisma.comment,
    [GraphQLModelType.Email]: (prisma: PrismaType) => prisma.email,
    [GraphQLModelType.Member]: (prisma: PrismaType) => prisma.organization_users,
    [GraphQLModelType.Node]: (prisma: PrismaType) => prisma.node,
    [GraphQLModelType.Organization]: (prisma: PrismaType) => prisma.organization,
    [GraphQLModelType.Profile]: (prisma: PrismaType) => prisma.user,
    [GraphQLModelType.Project]: (prisma: PrismaType) => prisma.project,
    [GraphQLModelType.Report]: (prisma: PrismaType) => prisma.report,
    [GraphQLModelType.Resource]: (prisma: PrismaType) => prisma.resource,
    [GraphQLModelType.Role]: (prisma: PrismaType) => prisma.role,
    [GraphQLModelType.Routine]: (prisma: PrismaType) => prisma.routine,
    [GraphQLModelType.Standard]: (prisma: PrismaType) => prisma.standard,
    [GraphQLModelType.Star]: (prisma: PrismaType) => prisma.star,
    [GraphQLModelType.Tag]: (prisma: PrismaType) => prisma.tag,
    [GraphQLModelType.User]: (prisma: PrismaType) => prisma.user,
    [GraphQLModelType.Vote]: (prisma: PrismaType) => prisma.vote,
}

/**
 * Helper function for adding join tables between 
 * many-to-many relationship parents and children
 * @param obj - GraphQL-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const addJoinTablesHelper = (partial: PartialInfo, map: JoinMap | undefined): any => {
    if (!map) return partial;
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object, pad with the join table name
        if (partial[key]) {
            result[key] = { [value]: partial[key] };
        }
    }
    return {
        ...partial,
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
 * Helper function for converting creator GraphQL field to Prisma createdByUser/createdByOrganization fields
 */
export const removeCreatorField = (select: any): any => {
    return deconstructUnion(select, 'creator', [
        [GraphQLModelType.User, 'createdByUser'],
        [GraphQLModelType.Organization, 'createdByOrganization']
    ]);
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
 * Helper function for converting owner GraphQL field to Prisma user/organization fields
 */
export const removeOwnerField = (select: any): any => {
    return deconstructUnion(select, 'owner', [
        [GraphQLModelType.User, 'user'],
        [GraphQLModelType.Organization, 'organization']
    ]);
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
 * Adds "select" to the correct parts of an object to make it a Prisma select
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
 * Removes the "__typename" field recursively from a JSON-serializable object
 * @param obj - JSON-serializable object with possible __typename fields
 * @return obj without __typename fields
 */
const removeTypenames = (obj: { [x: string]: any }): { [x: string]: any } => {
    return JSON.parse(JSON.stringify(obj, (k, v) => (k === '__typename') ? undefined : v))
}

/**
 * Converts a GraphQL info object to a partial Prisma select. 
 * This is then passed into a model-specific converter to handle virtual/calculated fields,
 * unions, and other special cases.
 * and keys that map to typemappers for each possible relationship
 * @param info - GraphQL info object, or result of this function
 * @param relationshipMap - Map of relationship names to typenames
 */
export const toPartialSelect = (info: InfoType, relationshipMap: RelationshipMap): PartialInfo | undefined => {
    // Return undefined if info not set
    console.log('topartialselect start', info)
    if (!info) return undefined;
    // Find select fields in info object
    let select;
    const isGraphQLResolveInfo = info.hasOwnProperty('fieldNodes') && info.hasOwnProperty('returnType');
    if (isGraphQLResolveInfo) {
        select = resolveGraphQLInfo(JSON.parse(JSON.stringify(info)));
        console.log('topartialselect isgraphqlresolveinfo', select?.nodes?.loop);
    } else {
        console.log('is not graphql resolve info');
        select = info;
    }
    // If fields are in the shape of a paginated search query, convert to a Prisma select object
    if (select.hasOwnProperty('pageInfo') && select.hasOwnProperty('edges')) {
        select = select.edges.node;
    }
    // Inject __typename fields
    console.log('going to inject typenames', JSON.stringify(select))
    select = injectTypenames(select, relationshipMap);
    console.log('injected typenames', select)
    return select;
}

/**
 * Recursively injects __typename fields into a select object
 * @param select - GraphQL select object, partially converted without typenames
 * and keys that map to typemappers for each possible relationship
 * @param parentRelationshipMap - Relationship of last known parent
 * @param nestedFields - Array of nested fields accessed since last parent
 * @return select with __typename fields
 */
const injectTypenames = (select: { [x: string]: any }, parentRelationshipMap: RelationshipMap, nestedFields: string[] = []): PartialInfo => {
    console.log('in injectTypenames start', select, parentRelationshipMap, nestedFields);
    // Create result object
    let result: any = {};
    // Iterate over select object
    for (const [selectKey, selectValue] of Object.entries(select)) {
        // Skip __typename
        if (selectKey === '__typename') continue;
        // If value is not an object, just add to result
        if (typeof selectValue !== 'object') {
            result[selectKey] = selectValue;
            continue;
        }
        // If value is an object, recurse
        // Find nested value in parent relationship map, using nestedFields
        let nestedValue: any = parentRelationshipMap;
        for (const field of nestedFields) {
            console.log('in field of nestedfields loop', field, nestedValue);
            if (!_.isObject(nestedValue)) break;
            if (field in nestedValue) {
                nestedValue = (nestedValue as any)[field];
            }
        }
        console.log('got nested value a', nestedValue)
        if (nestedValue) nestedValue = nestedValue[selectKey];
        console.log('got nested value b', nestedValue)
        // If nestedValue is not an object, try to get its relationshipMap
        let relationshipMap;
        if (typeof nestedValue !== 'object') relationshipMap = FormatterMap[nestedValue]?.relationshipMap;
        // If relationship map found, this becomes the new parent
        if (relationshipMap) {
            // New parent found, so we recurse with nestFields removed
            result[selectKey] = injectTypenames(selectValue, relationshipMap, []);
        }
        else {
            // No relationship map found, so we recurse and add this key to the nestedFields
            result[selectKey] = injectTypenames(selectValue, parentRelationshipMap, [...nestedFields, selectKey]);
        }
    }
    // Add __typename field if known
    if (nestedFields.length === 0) result.__typename = parentRelationshipMap.__typename;
    return result;
}

/**
 * Helper function for creating a Prisma select object. 
 * If the select object is in the shape of a paginated search query, 
 * then it will be converted to a prisma select object.
 * @returns select object for Prisma operations
 */
export const selectHelper = (partial: PartialInfo): any => {
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified = selectToDB(partial);
    if (!_.isObject(modified)) return undefined;
    // Delete __typename fields
    modified = removeTypenames(modified);
    // Pad every relationship with "select"
    modified = padSelect(modified);
    console.log('selecthelper end', JSON.stringify(modified));
    return modified;
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
 * Converts a GraphQL union into a Prisma select.
 * All possible union fields which the user wants to select are in the info partial, but they are not separated by type.
 * This function performs a nested intersection between each union type's avaiable fields, and the fields which were 
 * requested by the user.
 * @param partial - partial select object
 * @param unionField - Name of the union field
 * @param relationshipTupes - array of [relationship type (i.e. how it appears in partial), name to convert the field to]
 * @returns partial select object with GraphQL union converted into Prisma relationship selects
 */
export const deconstructUnion = (partial: any, unionField: string, relationshipTuples: [GraphQLModelType, string][]): any => {
    let { [unionField]: unionData, ...rest } = partial;
    // Create result object
    let converted: { [x: string]: any } = {};
    // If field in partial is not an object, return partial unmodified
    if (!unionData) return partial;
    // Swap keys of union to match their prisma names
    for (const [relType, relName] of relationshipTuples) {
        // If union missing, skip
        if (!unionData.hasOwnProperty(relType)) continue;
        const currData = unionData[relType];
        converted[relName] = currData;
    }
    return { ...rest, ...converted };
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
    // Initialize result with a. If any of a's values are not arrays, make them arrays
    for (const [key, value] of Object.entries(a)) {
        if (Array.isArray(value)) {
            result[key] = value;
        } else {
            result[key] = [value];
        }
    }
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
    return result;
}

/**
 * Combines fields from a Prisma object with arbitrarily nested relationships
 * @param data GraphQL-shaped data, where each object contains at least an ID
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns [objectDict, selectFieldsDict], where objectDict is a dictionary of object arrays (sorted by type), 
 * and selectFieldsDict is a dictionary of select fields for each type (unioned if they show up in multiple places)
 */
const groupIdsByType = (data: { [x: string]: any }, partial: PartialInfo): [{ [x: string]: object[] }, { [x: string]: any }] => {
    // Skip if __typename not in partial
    if (!partial?.__typename) return [{}, {}];
    let objectIdsDict: { [x: string]: { [x: string]: any }[] } = {};
    // Select fields for each type. If a type appears twice, the fields are combined.
    let selectFieldsDict: { [x: string]: { [x: string]: { [x: string]: any } } } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // If __typename for key does not exist (i.e. it was not requested), skip
            if (!partial[key] || !(typeof partial[key] === 'object') || !(partial[key] as any)?.__typename) continue;
            const childPartial: PartialInfo = partial[key] as any;
            const childType: string = childPartial.__typename as string;
            selectFieldsDict[childType] = _.merge(selectFieldsDict[childType] ?? {}, childPartial);
            // Pass each element through groupSupplementsByType
            for (const v of value) {
                const [childObjectsDict, childSelectFieldsDict] = groupIdsByType(v, childPartial);
                objectIdsDict = mergeObjectTypeDict(objectIdsDict, childObjectsDict);
                selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
            }
        }
        // If value is an object (and not date), add it to the correct key in objectDict
        else if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            // If __typename for key does not exist (i.e. it was not requested), skip
            if (!partial[key] || !(typeof partial[key] === 'object') || !(partial[key] as any)?.__typename) continue;
            const childPartial: PartialInfo = partial[key] as any;
            const childType: string = childPartial.__typename as string;
            selectFieldsDict[childType] = _.merge(selectFieldsDict[childType] ?? {}, childPartial);
            // Pass value through groupSupplementsByType
            const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(value, childPartial);
            objectIdsDict = mergeObjectTypeDict(objectIdsDict, childObjectIdsDict);
            selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
        }
    }
    // Handle base case
    // Remove anything that's not a primitive from partial
    const partialPrimitives = Object.keys(partial)
        .filter(key => !(typeof partial[key] === 'object') || (partial[key] as any)?.__typename)
        .reduce((res: any, key) => (res[key] = partial[key], res), {});
    // Finally, add the base object to objectDict (primitives only) and selectFieldsDict
    const currType: string = partial.__typename;
    if (currType in objectIdsDict) {
        objectIdsDict[currType].push(data);
    } else {
        objectIdsDict[currType] = [data];
    }
    selectFieldsDict[currType] = _.merge(selectFieldsDict[currType] ?? {}, partialPrimitives);
    // Return objectDict and selectFieldsDict
    return [objectIdsDict, selectFieldsDict];
}

/**
 * Recombines objects from supplementalFields calls into shape that matches info
 * @param data Original, unsupplemented data, where each object has an ID
 * @param objectsById Dictionary of objects with supplemental fields added, where each value contains at least an ID
 * @returns data with supplemental fields added
 */
const combineSupplements = (data: { [x: string]: any }, objectsById: { [x: string]: any }) => {
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
    // Handle base case
    return _.merge(data, objectsById[data.id])
}

/**
 * Recursively checks object for a child object with the given ID
 * @param data Object to check
 * @param id ID to check for
 * @returns Object with ID if found, otherwise undefined
 */
const findObjectById = (data: { [x: string]: any }, id: string): any => {
    // If data is an array, check each element
    if (Array.isArray(data)) {
        return data.map((v: any) => findObjectById(v, id));
    }
    // If data is an object, check for ID and return if found
    else if (_.isObject(data)) {
        if ((data as any).id === id) return data;
        // If data has children, check them
        else if (Object.keys(data).length > 0) {
            for (const [key, value] of Object.entries(data)) {
                const result = findObjectById(value, id);
                if (result) return result;
            }
        }
    }
    // Otherwise, return undefined
    return undefined;
}

/**
 * Picks the ID field from a nested object
 * @param data Object to pick ID from
 * @returns Object with all fields except nexted objects/arrays and ID removed
 */
 const pickId = (data: { [x: string]: any }): { [x: string]: any } => {
    var result: { [x: string]: any } = {};
    Object.keys(data).forEach((key) => {
        if (key === 'id') {
            result[key] = data[key];
        } else if (_.isArray(data[key])) {
            result[key] = data[key].map((v: any) => pickId(v));
        } else if (_.isObject(data[key]) && !(Object.prototype.toString.call(data[key]) === '[object Date]')) {
            result[key] = pickId(data[key]);
        }
    });
    return result;
}

/**
 * Picks an object from a nested object, using the given ID
 * @param data Object array to pick from
 * @param id ID to pick
 * @returns Requested object with all its fields and children included
 */
const pickObjectById = (data: any, id: string): any => {
    // If data is an array, check each element
    if (_.isArray(data)) {
        for (const value of data) {
            const result = pickObjectById(value, id);
            if (result) return result;
        }
    }
    // If data is an object (and not a date), check for ID and return if found
    else if (_.isObject(data) && !(Object.prototype.toString.call(data) === '[object Date]')) {
        if ((data as any).id === id) return data; // Base case
        // If ID doesn't match, check children
        for (const value of Object.values(data)) {
            const result = pickObjectById(value, id);
            if (result) return result
        }
    }
    // Otherwise, return undefined
    return undefined;
}

/**
 * Adds supplemental fields to the select object, and all of its relationships (and their relationships, etc.)
 * Groups objects types together, so database is called only once for each type.
 * @param prisma Prisma client
 * @param userId Requesting user's ID
 * @param data Array of GraphQL-shaped data, where each object contains at least an ID
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns data array with supplemental fields added to each object
 */
export const addSupplementalFields = async (
    prisma: PrismaType,
    userId: string | null,
    data: { [x: string]: any }[],
    partial: PartialInfo,
): Promise<{ [x: string]: any }[]> => {
    console.log('addsupplementalfields start', partial.__typename, data);
    // Strip all fields which are not the ID or a child array/object from data
    const dataIds: { [x: string]: any}[] = data.map(pickId);
    console.log('data ids here', JSON.stringify(dataIds));
    // Group data IDs and select fields by type. This is needed to reduce the number of times 
    // the database is called, as we can query all objects of the same type at once
    let objectIdsDict: { [x: string]: { [x: string]: any }[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: any } } = {};
    for (const d of dataIds) {
        const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(d, partial);
        // Merge each array in childObjectIdsDict into objectIdsDict
        objectIdsDict = mergeObjectTypeDict(objectIdsDict, childObjectIdsDict);
        // Merge each object in childSelectFieldsDict into selectFieldsDict
        selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
    }

    // Dictionary to store objects by ID, instead of type. This is needed to combineSupplements
    const objectsById: { [x: string]: any } = {};
    console.log('objectIdsDict', partial.__typename, objectIdsDict); //TODO check here

    // Loop through each type in objectIdsDict
    for (const [type, ids] of Object.entries(objectIdsDict)) {
        // Find the data for each id in ids. Since the data parameter is an array,
        // we must loop through each element in it and call pickObjectById
        const objectData = ids.map((id: { [x: string]: any }) => {
            return pickObjectById(data, id.id);
        });
        console.log('object data324 ', type, objectData);
        // Now that we have the data for each object, we can add the supplemental fields
        if (type in FormatterMap) {
            const valuesWithSupplements = FormatterMap[type]?.addSupplementalFields
                ? await (FormatterMap[type] as any).addSupplementalFields(prisma, userId, objectData, selectFieldsDict[type])
                : objectData;
            console.log('called object addSupplementalFields', type, valuesWithSupplements);
            // Add each value to objectsById
            for (const v of valuesWithSupplements) {
                console.log('adding to objectsById', v.id);
                objectsById[v.id] = v;
            }
        }
    }

    console.log('going to recursive combineSupplements', objectsById); //TODO only 9ea8 Cardano tag shows up here
    let result = data.map(d => combineSupplements(d, objectsById));
    console.log('got result', partial.__typename, JSON.stringify(result));
    // Convert objectsById dictionary back into shape of data
    return result
}

/**
 * Shapes GraphQL info object to become a valid Prisma select
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Valid Prisma select object
 */
export const selectToDB = (partial: PartialInfo): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recursively call selectToDB
        if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            console.log('selectToDB recursion', key, value, selectToDB(value as PartialInfo));
            result[key] = selectToDB(value as PartialInfo);
        }
        // Otherwise, add key/value pair to result
        else {
            result[key] = value;
        }
    }
    // Handle base case
    const typename = partial?.__typename;
    console.log('will selectdb handle base case?', typename, partial)
    if (typename && typename in FormatterMap) {
        const formatter = FormatterMap[typename];
        // Remove calculated fields and/or add fields for calculating
        if (formatter.removeCalculatedFields) result = formatter.removeCalculatedFields(result);
        // Deconstruct unions
        if (formatter.deconstructUnions) result = formatter.deconstructUnions(result);
        // Add join tables
        if (formatter.addJoinTables) result = formatter.addJoinTables(result);
    }
    return result;
}

/**
 * Shapes Prisma model object to become a valid GraphQL model object
 * @param data Prisma object
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Valid GraphQL object
 */
export function modelToGraphQL<GraphQLModel>(data: { [x: string]: any }, partial: PartialInfo): GraphQLModel {
    // First convert data to usable shape
    const typename = partial?.__typename;
    if (typename && typename in FormatterMap) {
        const formatter = FormatterMap[typename];
        // Construct unions
        if (formatter.constructUnions) data = formatter.constructUnions(data);
        // Remove join tables
        if (formatter.removeJoinTables) data = formatter.removeJoinTables(data);
    }
    // Then loop through each key/value pair in data and call modelToGraphQL on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partial, skip
        if (!_.isObject(partial) || !(partial as any)[key]) continue;
        // If value is an array, call modelToGraphQL on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGraphQL
            data[key] = data[key].map((v: any) => modelToGraphQL(v, partial[key] as PartialInfo));
        }
        // If value is an object (and not date), call modelToGraphQL on it
        else if (_.isObject(value) && !(Object.prototype.toString.call(value) === '[object Date]')) {
            data[key] = modelToGraphQL(value, (partial as any)[key]);
        }
    }
    return data as any;
}

/**
 * Helper function for reading one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL find by ID input object
 * @param info GraphQL info object
 * @param model Business layer object
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel>(
    userId: string | null,
    input: FindByIdInput,
    info: InfoType,
    model: ModelBusinessLayer<GraphQLModel, any>,
): Promise<RecursivePartial<GraphQLModel>> {
    console.log('in readOneHelper');
    // Validate input
    if (!input.id) throw new CustomError(CODE.InvalidArgs, 'id is required');
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    console.log('in readonheler after topartialselect', JSON.stringify(partial))
    if (!partial) throw new CustomError(CODE.InternalError, 'Could not convert info to partial select');
    // Uses __typename to determine which Prisma object is being queried
    const objectType = partial.__typename;
    if (!objectType || !(objectType in PrismaMap)) {
        throw new CustomError(CODE.InternalError, `${objectType} not found`);
    }
    // Get the Prisma object
    let object = await PrismaMap[objectType](model.prisma).findUnique({ where: { id: input.id }, ...selectHelper(partial) });
    if (!object) throw new CustomError(CODE.NotFound, `${objectType} not found`);
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, partial) as RecursivePartial<GraphQLModel>;
    console.log('ffffffffff formatted', formatted);
    return (await addSupplementalFields(model.prisma, userId, [formatted], partial))[0] as RecursivePartial<GraphQLModel>;
}

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * @param userId ID of user making the request
 * @param input GraphQL search input object
 * @param info GraphQL info object
 * @param model Business layer object
 * @param additionalQueries Additional where clauses to apply to the search
 * @returns Paginated search result
 */
export async function readManyHelper<GraphQLModel, SearchInput extends SearchInputBase<any>>(
    userId: string | null,
    input: SearchInput,
    info: InfoType,
    model: ModelBusinessLayer<GraphQLModel, SearchInput>,
    additionalQueries?: { [x: string]: any },
): Promise<PaginatedSearchResult> {
    console.log('readmanyhelper start', info)
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    if (!partial) throw new CustomError(CODE.InternalError, 'Could not convert info to partial select');
    console.log('readmanyhelper partial info', partial, info)
    // Uses __typename to determine which Prisma object is being queried
    const objectType = partial.__typename;
    if (!objectType || !(objectType in PrismaMap)) throw new CustomError(CODE.InternalError, `${objectType} not found`);
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
    // Determine text search query
    const searchQuery = (input.searchString && model.getSearchStringQuery) ? model.getSearchStringQuery(input.searchString) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    const typeQuery = SearcherMap[objectType]?.customQueries ? (SearcherMap[objectType] as any).customQueries(input) : undefined;
    // Combine queries
    const where = { ...additionalQueries, ...idQuery, ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
    // Determine sort order
    const orderBy = model.getSortQuery ? model.getSortQuery(input.sortBy ?? model.defaultSort) : undefined;
    // Find requested search array
    const searchResults = await PrismaMap[objectType](model.prisma).findMany({
        where,
        orderBy,
        take: input.take ?? 20,
        skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        cursor: input.after ? {
            id: input.after
        } : undefined,
        ...selectHelper(partial)
    });
    // If there are results
    let paginatedResults: PaginatedSearchResult;
    if (searchResults.length > 0) {
        // Find cursor
        const cursor = searchResults[searchResults.length - 1].id;
        // Query after the cursor to check if there are more results
        const hasNextPage = await PrismaMap[objectType](model.prisma).findMany({
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
    // Return formatted for GraphQL
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    console.log('findmany nodes before modeltographql', formattedNodes);
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, partial as PartialInfo));
    console.log('findmany nodes before addSupplementalFields', formattedNodes);
    formattedNodes = await addSupplementalFields(model.prisma, userId, formattedNodes, partial);
    console.log('findmany nodes after addSupplementalFields', formattedNodes);
    return { pageInfo: paginatedResults.pageInfo, edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
}

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @param input Count metrics common to all models
 * @param model Business layer object
 * @param where Additional where clauses, in addition to the createdMetric and updatedMetric passed into input
 * @returns The number of matching objects
 */
export async function countHelper<CountInput extends CountInputBase>(input: CountInput, model: ModelBusinessLayer<any, any>, where?: { [x: string]: any }): Promise<number> {
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Count objects that match queries
    return await PrismaMap[model.relationshipMap.__typename as string](model.prisma).count({
        where: {
            ...where,
            ...createdQuery,
            ...updatedQuery,
        },
    });
}

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @param userId ID of user making the request
 * @param input GraphQL create input object
 * @param info GraphQL info object
 * @param model Business layer object
 * @returns GraphQL response object
 */
export async function createHelper<GraphQLModel>(
    userId: string | null,
    input: any,
    info: InfoType,
    model: ModelBusinessLayer<GraphQLModel, any>,
): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object');
    if (!model.cud) throw new CustomError(CODE.InternalError, 'Model does not support create');
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    const partial = toPartialSelect(info, model.relationshipMap);
    if (!partial) throw new CustomError(CODE.InternalError, 'Could not convert info to partial select');
    const boop = await model.cud({ partial, userId, createMany: [input] });
    const { created } = boop
    if (created && created.length > 0) {
        return (await addSupplementalFields(model.prisma, userId, created, partial))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown);
}

/**
 * Helper function for updating one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL update input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @param prisma Prisma client
 * @returns GraphQL response object
 */
export async function updateHelper<GraphQLModel>(
    userId: string | null,
    input: any,
    info: InfoType,
    model: ModelBusinessLayer<GraphQLModel, any>
): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object');
    if (!model.cud) throw new CustomError(CODE.InternalError, 'Model does not support update');
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    if (!partial) throw new CustomError(CODE.InternalError, 'Could not convert info to partial select');
    const { updated } = await model.cud({ partial, userId, updateMany: [input] });
    if (updated && updated.length > 0) {
        return (await addSupplementalFields(model.prisma, userId, updated, partial))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown);
}

/**
 * Helper function for deleting one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper(
    userId: string | null,
    input: any,
    model: ModelBusinessLayer<any, any>,
): Promise<Success> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete object');
    if (!model.cud) throw new CustomError(CODE.InternalError, 'Model does not support delete');
    const { deleted } = await model.cud({ partial: {}, userId, deleteMany: [input.id] });
    return { success: Boolean((deleted as any)?.count > 0) };
}

/**
 * Helper function for deleting many of the same object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper(
    userId: string | null,
    input: any,
    model: ModelBusinessLayer<any, any>,
): Promise<Count> {
    if (!userId) throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete objects');
    if (!model.cud) throw new CustomError(CODE.InternalError, 'Model does not support delete');
    const { deleted } = await model.cud({ partial: {}, userId, deleteMany: [input.id] });
    if (!deleted) throw new CustomError(CODE.ErrorUnknown);
    return deleted
}