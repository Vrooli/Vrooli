// Components for providing basic functionality to model objects
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, PageInfo, Success, TimeFrame } from '../../schema/types';
import { PrismaType, RecursivePartial } from '../../types';
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
import { CustomError } from '../../error';
import { CODE, ViewFor } from '@local/shared';
import { profileFormatter } from './profile';
import { memberFormatter } from './member';
import { resolveGraphQLInfo } from '../../utils';
import { inputItemFormatter } from './inputItem';
import { outputItemFormatter } from './outputItem';
import { resourceListFormatter, resourceListSearcher } from './resourceList';
import { tagHiddenFormatter } from './tagHidden';
import { Log, LogType } from '../../models/nosql';
import { genErrorCode } from '../../logger';
import { viewFormatter, ViewModel } from './view';
import { runFormatter, runSearcher } from './run';
const { isObject } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

export enum GraphQLModelType {
    Comment = 'Comment',
    Email = 'Email',
    Handle = 'Handle',
    InputItem = 'InputItem',
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
    Report = 'Report',
    Resource = 'Resource',
    ResourceList = 'ResourceList',
    Role = 'Role',
    Routine = 'Routine',
    Run = 'Run',
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

export type RelationshipMap = { 
    __typename: GraphQLModelType;
    [relationshipName: string]: GraphQLModelType | { [fieldName: string]: GraphQLModelType } 
};

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
     * Add _count fields
     */
    addCountFields?: (partial: { [x: string]: any }) => any;
    /**
     * Remove _count fields
     */
    removeCountFields?: (data: { [x: string]: any }) => any;
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

export interface ValidateMutationsInput<Create, Update> {
    userId: string | null,
    createMany?: Create[] | null | undefined,
    updateMany?: { where: { id: string }, data: Update }[] | null | undefined,
    deleteMany?: string[] | null | undefined,
}

export interface CUDInput<Create, Update> {
    userId: string | null,
    createMany?: Create[] | null | undefined,
    updateMany?: { where: { id: string }, data: Update }[] | null | undefined,
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

export const FormatterMap: { [key in GraphQLModelType]?: FormatConverter<any> } = {
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
    [GraphQLModelType.Run]: runFormatter(),
    [GraphQLModelType.Standard]: standardFormatter(),
    [GraphQLModelType.Star]: starFormatter(),
    [GraphQLModelType.Tag]: tagFormatter(),
    [GraphQLModelType.TagHidden]: tagHiddenFormatter(),
    [GraphQLModelType.User]: userFormatter(),
    [GraphQLModelType.Vote]: voteFormatter(),
    [GraphQLModelType.View]: viewFormatter(),
}

export const SearcherMap: { [key in GraphQLModelType]?: Searcher<any> } = {
    [GraphQLModelType.Comment]: commentSearcher(),
    // 'Member': memberSearcher(),TODO create searchers for all these
    [GraphQLModelType.Organization]: organizationSearcher(),
    [GraphQLModelType.Project]: projectSearcher(),
    [GraphQLModelType.Report]: reportSearcher(),
    [GraphQLModelType.Resource]: resourceSearcher(),
    [GraphQLModelType.ResourceList]: resourceListSearcher(),
    [GraphQLModelType.Routine]: routineSearcher(),
    [GraphQLModelType.Run]: runSearcher(),
    [GraphQLModelType.Standard]: standardSearcher(),
    // 'Star': starSearcher(),
    [GraphQLModelType.Tag]: tagSearcher(),
    [GraphQLModelType.User]: userSearcher(),
    // 'Vote': voteSearcher(),
}

export const PrismaMap: { [key in GraphQLModelType]?: (prisma: PrismaType) => any } = {
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
    [GraphQLModelType.Run]: (prisma: PrismaType) => prisma.run,
    [GraphQLModelType.Standard]: (prisma: PrismaType) => prisma.standard,
    [GraphQLModelType.RunStep]: (prisma: PrismaType) => prisma.run_step,
    [GraphQLModelType.Star]: (prisma: PrismaType) => prisma.star,
    [GraphQLModelType.Tag]: (prisma: PrismaType) => prisma.tag,
    [GraphQLModelType.User]: (prisma: PrismaType) => prisma.user,
    [GraphQLModelType.Vote]: (prisma: PrismaType) => prisma.vote,
    [GraphQLModelType.View]: (prisma: PrismaType) => prisma.view,
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
export const addCountFieldsHelper = (obj: any, map: CountMap | undefined): any => {
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
export const removeCountFieldsHelper = (obj: any, map: CountMap): any => {
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
export const padSelect = (fields: { [x: string]: any }): { [x: string]: any } => {
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
    if (!info) return undefined;
    // Find select fields in info object
    let select;
    const isGraphQLResolveInfo = info.hasOwnProperty('fieldNodes') && info.hasOwnProperty('returnType');
    if (isGraphQLResolveInfo) {
        select = resolveGraphQLInfo(JSON.parse(JSON.stringify(info)));
    } else {
        select = info;
    }
    // If fields are in the shape of a paginated search query, convert to a Prisma select object
    if (select.hasOwnProperty('pageInfo') && select.hasOwnProperty('edges')) {
        select = select.edges.node;
    }
    // Inject __typename fields
    select = injectTypenames(select, relationshipMap);
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
        let nestedValue: GraphQLModelType | Partial<RelationshipMap> | undefined = parentRelationshipMap;
        for (const field of nestedFields) {
            if (!_.isObject(nestedValue)) break;
            if (field in nestedValue) {
                nestedValue = (nestedValue as any)[field];
            }
        }
        if (typeof nestedValue === 'object') nestedValue = nestedValue[selectKey];
        // If nestedValue is not an object, try to get its relationshipMap
        let relationshipMap;
        if (nestedValue !== undefined && typeof nestedValue !== 'object') relationshipMap = FormatterMap[nestedValue]?.relationshipMap;
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
    connect?: { id: string }[],
    disconnect?: { id: string }[],
    delete?: { id: string }[],
    create?: { [x: string]: any }[],
    update?: { where: { id: string }, data: { [x: string]: any } }[],
} => {
    // Determine valid operations, and remove operations that should be excluded
    let ops = isAdd ? [RelationshipTypes.connect, RelationshipTypes.create] : Object.values(RelationshipTypes);
    ops = _.difference(ops, relExcludes)
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    for (const [key, value] of Object.entries(data)) {
        if (value === null || value === undefined) continue;
        // Skip if not matching relationship or not a valid operation
        if (!key.startsWith(relationshipName) || !ops.some(o => key.toLowerCase().endsWith(o))) continue;
        // Determine operation
        const currOp = key.replace(relationshipName, '').toLowerCase();
        // TODO handle soft delete
        // Add operation to result object
        const shapedData = shapeRelationshipData(value, fieldExcludes);
        converted[currOp] = Array.isArray(converted[currOp]) ? [...converted[currOp], ...shapedData] : shapedData;
    };
    // Connects, diconnects, and deletes must be shaped in the form of { id: '123' } (i.e. no other fields)
    if (Array.isArray(converted.connect) && converted.connect.length > 0) converted.connect = converted.connect.map((e: { [x: string]: any }) => ({ id: e.id }));
    if (Array.isArray(converted.disconnect) && converted.disconnect.length > 0) converted.disconnect = converted.disconnect.map((e: { [x: string]: any }) => ({ id: e.id }));
    if (Array.isArray(converted.delete) && converted.delete.length > 0) converted.delete = converted.delete.map((e: { [x: string]: any }) => ({ id: e.id }));
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(converted.update) && converted.update.length > 0) {
        converted.update = converted.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    return converted;
}

export interface JoinRelationshipToPrismaArgs extends RelationshipToPrismaArgs {
    joinFieldName: string, // e.g. organization.tags.tag => 'tag'
    uniqueFieldName: string, // e.g. organization.tags.tag => 'organization_tags_taggedid_tagid_unique'
    childIdFieldName: string, // e.g. organization.tags.tag => 'tagId'
    parentIdFieldName: string, // e.g. organization.tags.tag => 'taggedId'
    parentId: string | null, // Only needed if not a create
}

/**
 * Converts the result of relationshipToPrisma to apply to a many-to-many relationship 
 * (i.e. uses a join table).
 * NOTE: Does not differentiate between a disconnect and a delete. How these are handled is determined by 
 * the database cascading.
 * NOTE: Can only update, disconnect, or delete if isAdd is false.
 * @param data The data to convert
 * @param joinFieldName The name of the field in the join table associated with the child object
 * @param uniqueFieldName The name of the unique field in the join table
 * @param childIdFieldName The name of the field in the join table associated with the child object
 * @param parentIdFieldName The name of the field in the join table associated with the parent object
 * @param relationshipName The name of the relationship to convert (since data may contain irrelevant fields)
 * @param isAdd True if data is being converted for an add operation. This limits the prisma operations to only "connect" and "create"
 * @param fieldExcludes Fields to exclude from the conversion
 * @param relExcludes Relationship types to exclude from the conversion
 * @param softDelete True if deletes should be converted to soft deletes
 */
export const joinRelationshipToPrisma = ({
    data,
    joinFieldName,
    uniqueFieldName,
    childIdFieldName,
    parentIdFieldName,
    parentId,
    relationshipName,
    isAdd,
    fieldExcludes = [],
    relExcludes = [],
    softDelete = false
}: JoinRelationshipToPrismaArgs): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    // Call relationshipToPrisma to get join data used for one-to-many relationships
    const normalJoinData = relationshipToPrisma({ data, relationshipName, isAdd, fieldExcludes, relExcludes, softDelete })
    // Convert this to support a join table
    if (normalJoinData.hasOwnProperty('connect')) {
        // ex: create: [ { tag: { connect: { id: 'asdf' } } } ] <-- A join table always creates on connects
        for (const id of (normalJoinData?.connect ?? [])) {
            const curr = { [joinFieldName]: { connect: id } };
            converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('disconnect')) {
        // delete: [ { organization_tags_taggedid_tagid_unique: { tagId: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
        for (const id of (normalJoinData?.disconnect ?? [])) {
            const curr = { [uniqueFieldName]: { [childIdFieldName]: id.id, [parentIdFieldName]: parentId } };
            converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('delete')) {
        // delete: [ { organization_tags_taggedid_tagid_unique: { tagId: 'asdf', taggedId: 'fdas' } } ]
        for (const id of (normalJoinData?.delete ?? [])) {
            const curr = { [uniqueFieldName]: { [childIdFieldName]: id.id, [parentIdFieldName]: parentId } };
            converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('create')) {
        // ex: create: [ { tag: { create: { id: 'asdf' } } } ]
        for (const id of (normalJoinData?.create ?? [])) {
            const curr = { [joinFieldName]: { create: id } };
            converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('update')) {
        // ex: update: [{ 
        //         where: { organization_tags_taggedid_tagid_unique: { tagId: 'asdf', taggedId: 'fdas' } },
        //         data: { tag: { update: { tag: 'fdas', } } }
        //     }]
        for (const data of (normalJoinData?.update ?? [])) {
            const curr = {
                where: { [uniqueFieldName]: { [childIdFieldName]: data.where.id, [parentIdFieldName]: parentId } },
                data: { [joinFieldName]: { update: data.data } }
            };
            converted.update = Array.isArray(converted.update) ? [...converted.update, curr] : [curr];
        }
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
 * Counts the "similarity" of two objects. The more fields that match, the higher the score.
 * @returns Total number of matching fields, or 0
 */
const countSubset = (obj1: any, obj2: any): number => {
    let count = 0;
    if (obj1 === null || typeof obj1 !== 'object' || obj2 === null || typeof obj2 !== 'object') return count;
    for (const key of Object.keys(obj1)) {
        // If union, add greatest count
        if (key[0] === key[0].toUpperCase()) {
            let highestCount = 0;
            for (const unionType of Object.keys(obj2)) {
                const currCount = countSubset(obj1[key], unionType);
                if (currCount > highestCount) highestCount = currCount;
            }
            count += highestCount;
        }
        if (Array.isArray((obj1)[key])) {
            // Only add count for one object in array
            if (obj1[key].length > 0) {
                count += countSubset(obj1[key][0], obj2[key]);
            }
        }
        else if (typeof (obj1)[key] === 'object') {
            count += countSubset(obj1[key], obj2[key]);
        }
        else count++
    }
    return count;
}

/**
 * Combines fields from a Prisma object with arbitrarily nested relationships
 * @param data GraphQL-shaped data, where each object contains at least an ID
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns [objectDict, selectFieldsDict], where objectDict is a dictionary of object arrays (sorted by type), 
 * and selectFieldsDict is a dictionary of select fields for each type (unioned if they show up in multiple places)
 */
const groupIdsByType = (data: { [x: string]: any }, partial: PartialInfo): [{ [x: string]: string[] }, { [x: string]: any }] => {
    if (!data || !partial) return [{}, {}];
    let objectIdsDict: { [x: string]: string[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: { [x: string]: any } } } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        let childPartial: PartialInfo = partial[key] as any;
        if (childPartial)
            // If every key in childPartial starts with a capital letter, then it is a union.
            // In this case, we must determine which union to use based on the shape of value
            if (_.isObject(childPartial) && Object.keys(childPartial).every(k => k[0] === k[0].toUpperCase())) {
                // Find the union type which matches the shape of value, using countSubset
                let highestCount = 0;
                let highestType: string = '';
                for (const unionType of Object.keys(childPartial)) {
                    const currCount = countSubset(value, childPartial[unionType]);
                    if (currCount > highestCount) {
                        highestCount = currCount;
                        highestType = unionType;
                    }
                }
                // If no union type matches, skip
                if (!highestType) continue;
                // If union type, update child partial
                childPartial = childPartial[highestType] as PartialInfo;
            }
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through groupSupplementsByType
            for (const v of value) {
                const [childObjectsDict, childSelectFieldsDict] = groupIdsByType(v, childPartial);
                for (const [childType, childObjects] of Object.entries(childObjectsDict)) {
                    objectIdsDict[childType] = objectIdsDict[childType] ?? [];
                    objectIdsDict[childType].push(...childObjects);
                }
                selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
            }
        }
        // If value is an object (and not date), add it to the correct key in objectDict
        else if (isRelationshipObject(value)) {
            // Pass value through groupIdsByType
            const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(value, childPartial);
            for (const [childType, childObjects] of Object.entries(childObjectIdsDict)) {
                objectIdsDict[childType] = objectIdsDict[childType] ?? [];
                objectIdsDict[childType].push(...childObjects);
            }
            selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
        }
        else if (key === 'id' && partial.__typename) {
            // Add to objectIdsDict
            const type: string = partial.__typename;
            objectIdsDict[type] = objectIdsDict[type] ?? [];
            objectIdsDict[type].push(value);
        }
    }
    // Add keys to selectFieldsDict
    const currType = partial?.__typename;
    if (currType) {
        selectFieldsDict[currType] = _.merge(selectFieldsDict[currType] ?? {}, partial);
    }
    // Return objectDict and selectFieldsDict
    return [objectIdsDict, selectFieldsDict];
}

/**
 * Determines if an object is a relationship object, and not an array of relationship objects.
 * @param obj - object to check
 * @returns true if obj is a relationship object, false otherwise
 */
const isRelationshipObject = (obj: any): boolean => _.isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]';

/**
 * Determines if an object is an array of relationship objects, and not a relationship object.
 * @param obj - object to check
 * @returns true if obj is an array of relationship objects, false otherwise
 */
const isRelationshipArray = (obj: any): boolean => Array.isArray(obj) && obj.every(e => isRelationshipObject(e));

/**
 * Recombines objects returned from calls to supplementalFields into shape that matches info
 * @param data Original, unsupplemented data, where each object has an ID
 * @param objectsById Dictionary of objects with supplemental fields added, where each value contains at least an ID
 * @returns data with supplemental fields added
 */
const combineSupplements = (data: { [x: string]: any }, objectsById: { [x: string]: any }) => {
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through combineSupplements
            result[key] = data[key].map((v: any) => combineSupplements(v, objectsById));
        }
        // If value is an object (and not a date), add it to the correct key in objectDict
        else if (isRelationshipObject(value)) {
            // Pass value through combineSupplements
            result[key] = combineSupplements(value, objectsById);
        }
    }
    // Handle base case
    return _.merge(result, objectsById[data.id])
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
            const idArray = data[key].map((v: any) => pickId(v));
            if (idArray.length > 0) result[key] = idArray;
        } else if (isRelationshipObject(data[key])) {
            const idObject = pickId(data[key]);
            if (Object.keys(idObject).length > 0) result[key] = idObject;
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
    else if (isRelationshipObject(data)) {
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
    data: ({ [x: string]: any } | null | undefined)[],
    partial: PartialInfo | PartialInfo[],
): Promise<{ [x: string]: any }[]> => {
    // Group data IDs and select fields by type. This is needed to reduce the number of times 
    // the database is called, as we can query all objects of the same type at once
    let objectIdsDict: { [x: string]: string[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: any } } = {};
    for (let i = 0; i < data.length; i++) {
        const currData = data[i];
        const currPartial = Array.isArray(partial) ? partial[i] : partial;
        if (!currData || !currPartial) continue;
        const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(currData, currPartial);
        // Merge each array in childObjectIdsDict into objectIdsDict
        for (const [childType, childObjects] of Object.entries(childObjectIdsDict)) {
            objectIdsDict[childType] = objectIdsDict[childType] ?? [];
            objectIdsDict[childType].push(...childObjects);
        }
        // Merge each object in childSelectFieldsDict into selectFieldsDict
        selectFieldsDict = _.merge(selectFieldsDict, childSelectFieldsDict);
    }

    // Dictionary to store objects by ID, instead of type. This is needed to combineSupplements
    const objectsById: { [x: string]: any } = {};

    // Loop through each type in objectIdsDict
    for (const [type, ids] of Object.entries(objectIdsDict)) {
        // Find the data for each id in ids. Since the data parameter is an array,
        // we must loop through each element in it and call pickObjectById
        const objectData = ids.map((id: string) => pickObjectById(data, id));
        // Now that we have the data for each object, we can add the supplemental fields
        if (type in FormatterMap) {
            const valuesWithSupplements = FormatterMap[type as keyof typeof FormatterMap]?.addSupplementalFields
                ? await (FormatterMap[type as keyof typeof FormatterMap] as any).addSupplementalFields(prisma, userId, objectData, selectFieldsDict[type])
                : objectData;
            // Add each value to objectsById
            for (const v of valuesWithSupplements) {
                objectsById[v.id] = v;
            }
        }
    }
    // Convert objectsById dictionary back into shape of data
    let result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, objectsById));
    return result
}

/**
 * Combines addSupplementalFields calls for multiple object types
 * @param data Array of arrays, where each array is a list of the same object type queried from the database
 * @param partials Array of PartialInfos, in the same order as data arrays
 * @param keys Keys to associate with each data array
 * @param userId Requesting user's ID
 * @param prisma Prisma client
 * @returns Object with keys equal to objectTypes, and values equal to arrays of objects with supplemental fields added
 */
export const addSupplementalFieldsMultiTypes = async (
    data: { [x: string]: any }[][],
    partials: PartialInfo[],
    keys: string[],
    userId: string | null,
    prisma: PrismaType,
): Promise<{ [x: string]: any[] }> => {
    // Flatten data array
    const combinedData = _.flatten(data);
    // Create an array of partials, that match the data array
    let combinedPartials: PartialInfo[] = [];
    for (let i = 0; i < data.length; i++) {
        const currPartial = partials[i];
        // Push partial for each data array
        for (let j = 0; j < data[i].length; j++) {
            combinedPartials.push(currPartial);
        }
    }
    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(prisma, userId, combinedData, combinedPartials);
    // Convert combinedResult into object with keys equal to objectTypes, and values equal to arrays of those types
    const formatted: { [y: string]: any[] } = {};
    let start = 0;
    for (let i = 0; i < keys.length; i++) {
        const currKey = keys[i];
        const end = start + data[i].length;
        formatted[currKey] = combinedResult.slice(start, end);
        start = end;
    }
    return formatted;
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
        if (isRelationshipObject(value)) {
            result[key] = selectToDB(value as PartialInfo);
        }
        // Otherwise, add key/value pair to result
        else {
            result[key] = value;
        }
    }
    // Handle base case
    const type: string | undefined = partial?.__typename;
    if (type !== undefined && type in FormatterMap) {
        const formatter: FormatConverter<any> = FormatterMap[type as keyof typeof FormatterMap] as FormatConverter<any>;
        if (formatter.removeCalculatedFields) result = formatter.removeCalculatedFields(result);
        if (formatter.deconstructUnions) result = formatter.deconstructUnions(result);
        if (formatter.addJoinTables) result = formatter.addJoinTables(result);
        if (formatter.addCountFields) result = formatter.addCountFields(result);
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
    const type: string | undefined = partial?.__typename;
    if (type !== undefined && type in FormatterMap) {
        const formatter: FormatConverter<any> = FormatterMap[type as keyof typeof FormatterMap] as FormatConverter<any>;
        if (formatter.constructUnions) data = formatter.constructUnions(data);
        if (formatter.removeJoinTables) data = formatter.removeJoinTables(data);
        if (formatter.removeCountFields) data = formatter.removeCountFields(data);
    }
    // Remove top-level union from partial, if necessary
    // If every key starts with a capital letter, it's a union
    if (Object.keys(partial).every(k => k[0] === k[0].toUpperCase())) {
        // Find the union type which matches the shape of value, using countSubset
        let highestCount = 0;
        let highestType: string = '';
        for (const unionType of Object.keys(partial)) {
            const currCount = countSubset(data, partial[unionType]);
            if (currCount > highestCount) {
                highestCount = currCount;
                highestType = unionType;
            }
        }
        if (highestType) partial = partial[highestType] as PartialInfo;
    }
    // Then loop through each key/value pair in data and call modelToGraphQL on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partial, check if union
        if (!_.isObject(partial) || !(partial as any)[key]) continue;
        // If value is an array, call modelToGraphQL on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGraphQL
            data[key] = data[key].map((v: any) => modelToGraphQL(v, partial[key] as PartialInfo));
        }
        // If value is an object (and not date), call modelToGraphQL on it
        else if (isRelationshipObject(value)) {
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
    // Validate input
    if (!input.id)
        throw new CustomError(CODE.InvalidArgs, 'id is required', { code: genErrorCode('0019') });
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    if (!partial)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0020') });
    // Uses __typename to determine which Prisma object is being queried
    const objectType: string | undefined = partial.__typename;
    if (objectType === undefined || !(objectType in PrismaMap)) {
        throw new CustomError(CODE.InternalError, `${objectType} missing in PrismaMap`, { code: genErrorCode('0021') });
    }
    // Get the Prisma object
    let object = await (PrismaMap[objectType as keyof typeof PrismaMap] as any)(model.prisma).findUnique({ where: { id: input.id }, ...selectHelper(partial) });
    if (!object)
        throw new CustomError(CODE.NotFound, `${objectType} not found`, { code: genErrorCode('0022') });
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, partial) as RecursivePartial<GraphQLModel>;
    // If logged in and object has view count, handle it
    if (userId && objectType in ViewFor) {
        console.log('adding to view count');
        ViewModel(model.prisma).view(userId, { forId: input.id, title: '', viewFor: objectType as any }); //TODO add title, which requires user's language
    }
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
 * @param addSupplemental Decides if queried data should be called. Defaults to true. 
 * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
 * later in one call
 * @returns Paginated search result
 */
export async function readManyHelper<GraphQLModel, SearchInput extends SearchInputBase<any>>(
    userId: string | null,
    input: SearchInput,
    info: InfoType,
    model: ModelBusinessLayer<GraphQLModel, SearchInput>,
    additionalQueries?: { [x: string]: any },
    addSupplemental = true,
): Promise<PaginatedSearchResult> {
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    if (!partial)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
    // Uses __typename to determine which Prisma object is being queried
    const objectType: string | undefined = partial.__typename;
    if (objectType === undefined || !(objectType in PrismaMap))
        throw new CustomError(CODE.InternalError, `${objectType} not found in PrismaMap`, { code: genErrorCode('0024') });
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: input.ids } }) : undefined;
    // Determine text search query
    const searchQuery = (input.searchString && model.getSearchStringQuery) ? model.getSearchStringQuery(input.searchString) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    let typeQuery: any = undefined;
    if (objectType in SearcherMap && SearcherMap[objectType as keyof typeof SearcherMap]?.customQueries) {
        typeQuery = (SearcherMap[objectType as keyof typeof SearcherMap] as any).customQueries(input);
    }
    // Combine queries
    const where = { ...additionalQueries, ...idQuery, ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
    // Determine sort order
    const orderBy = model.getSortQuery ? model.getSortQuery(input.sortBy ?? model.defaultSort) : undefined;
    // Find requested search array
    const searchResults = await (PrismaMap[objectType as keyof typeof PrismaMap] as any)(model.prisma).findMany({
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
        const hasNextPage = await (PrismaMap[objectType as keyof typeof PrismaMap] as any)(model.prisma).findMany({
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
    // If not adding supplemental fields, return the paginated results
    if (!addSupplemental) return paginatedResults;
    // Return formatted for GraphQL
    let formattedNodes = paginatedResults.edges.map(({ node }) => node);
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, partial as PartialInfo));
    formattedNodes = await addSupplementalFields(model.prisma, userId, formattedNodes, partial);
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
    // Check if model is supported
    const type: string = model.relationshipMap.__typename;
    if (!type || !(type in PrismaMap))
        throw new CustomError(CODE.InternalError, `${type} not found in PrismaMap`, { code: genErrorCode('0183') });
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Count objects that match queries
    return await (PrismaMap[type as keyof typeof PrismaMap] as any)(model.prisma).count({
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
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0025') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support create', { code: genErrorCode('0026') });
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    const partial = toPartialSelect(info, model.relationshipMap);
    if (!partial)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0027') });
    const cudResult = await model.cud({ partial, userId, createMany: [input] });
    const { created } = cudResult;
    if (created && created.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partial.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = created.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Create,
                object1Type: objectType,
                object1Id: c.id,
            }));
            console.log('before log createhelper', JSON.stringify(logs), '\n\n')
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs)
        }
        return (await addSupplementalFields(model.prisma, userId, created, partial))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in createHelper', { code: genErrorCode('0028') });
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
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0029') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support update', { code: genErrorCode('0030') });
    // Partially convert info type so it is easily usable (i.e. in prisma mutation shape, but with __typename and without padded selects)
    let partial = toPartialSelect(info, model.relationshipMap);
    if (!partial)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0031') });
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: { id: input.id }, data: input };
    const { updated } = await model.cud({ partial, userId, updateMany: [shapedInput] });
    if (updated && updated.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partial.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = updated.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Update,
                object1Type: objectType,
                object1Id: c.id,
            }));
            console.log('before log updatehelper', JSON.stringify(logs), '\n\n')
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs)
        }
        return (await addSupplementalFields(model.prisma, userId, updated, partial))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in updateHelper', { code: genErrorCode('0032') });
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
    input: DeleteOneInput,
    model: ModelBusinessLayer<any, any>,
): Promise<Success> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete object', { code: genErrorCode('0033') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0034') });
    const { deleted } = await model.cud({ partial: {}, userId, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = model.relationshipMap.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            console.log('before log deleteone')
            // No need to await this, since it is not needed for the response
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Delete,
                object1Type: objectType,
                object1Id: input.id,
            })
        }
        return { success: true }
    }
    return { success: false };
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
    input: DeleteManyInput,
    model: ModelBusinessLayer<any, any>,
): Promise<Count> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete objects', { code: genErrorCode('0035') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0036') });
    const { deleted } = await model.cud({ partial: {}, userId, deleteMany: input.ids });
    if (!deleted)
        throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in deleteManyHelper', { code: genErrorCode('0037') });
    // If organization, project, routine, or standard, log for stats
    const objectType = model.relationshipMap.__typename;
    if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
        const logs = input.ids.map((id: string) => ({
            timestamp: Date.now(),
            userId: userId,
            action: LogType.Delete,
            object1Type: objectType,
            object1Id: id,
        }));
        console.log('before log deleteManyHelper', JSON.stringify(logs), '\n\n')
        // No need to await this, since it is not needed for the response
        Log.collection.insertMany(logs)
    }
    return deleted
}