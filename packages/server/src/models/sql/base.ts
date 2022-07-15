// Components for providing basic functionality to model objects
import { CopyInput, CopyType, Count, DeleteManyInput, DeleteOneInput, FindByIdOrHandleInput, ForkInput, PageInfo, Success, TimeFrame } from '../../schema/types';
import { PrismaType, RecursivePartial } from '../../types';
import { GraphQLResolveInfo } from 'graphql';
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
import { CODE, isObject, ViewFor } from '@local/shared';
import { profileFormatter } from './profile';
import { memberFormatter } from './member';
import { resolveGraphQLInfo } from '../../utils';
import { inputItemFormatter } from './inputItem';
import { outputItemFormatter } from './outputItem';
import { resourceListFormatter, resourceListSearcher } from './resourceList';
import { tagHiddenFormatter } from './tagHidden';
import { Log, LogType } from '../../models/nosql';
import { genErrorCode, logger, LogLevel } from '../../logger';
import { viewFormatter, ViewModel } from './view';
import { runFormatter, runSearcher } from './run';
import pkg from 'lodash';
const { difference, flatten, merge } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

export enum GraphQLModelType {
    Comment = 'Comment',
    Copy = 'Copy',
    Email = 'Email',
    Fork = 'Fork',
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
    cud?({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<any, any>): Promise<CUDResult<GraphQLModel>>
    duplicate?({ userId, objectId, isFork, createCount }: DuplicateInput): Promise<DuplicateResult<GraphQLModel>>
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
    removeCalculatedFields?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
    /**
     * Convert database fields to GraphQL union types
     */
    constructUnions?: (data: { [x: string]: any }) => any;
    /**
     * Convert GraphQL unions to database fields
     */
    deconstructUnions?: (partial: PartialGraphQLInfo | PartialPrismaSelect) => any;
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
        partial: PartialGraphQLInfo,
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
 * Determines if an object is a relationship object, and not an array of relationship objects.
 * @param obj - object to check
 * @returns true if obj is a relationship object, false otherwise
 */
const isRelationshipObject = (obj: any): boolean => isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]';

/**
 * Determines if an object is an array of relationship objects, and not a relationship object.
 * @param obj - object to check
 * @returns true if obj is an array of relationship objects, false otherwise
 */
const isRelationshipArray = (obj: any): boolean => Array.isArray(obj) && obj.every(e => isRelationshipObject(e));

/**
 * Helper function for adding join tables between 
 * many-to-many relationship parents and children
 * @param obj - GraphQL-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const addJoinTablesHelper = (partialInfo: PartialGraphQLInfo, map: JoinMap | undefined): any => {
    if (!map) return partialInfo;
    // Create result object
    let result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object, pad with the join table name
        if (partialInfo[key]) {
            // Otherwise, pad
            result[key] = { [value]: partialInfo[key] };
        }
    }
    return {
        ...partialInfo,
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
    // Iterate over count map
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
 * Recursively injects __typename fields into a select object
 * @param select - GraphQL select object, partially converted without typenames
 * and keys that map to typemappers for each possible relationship
 * @param parentRelationshipMap - Relationship of last known parent
 * @param nestedFields - Array of nested fields accessed since last parent
 * @return select with __typename fields
 */
const injectTypenames = (select: { [x: string]: any }, parentRelationshipMap: RelationshipMap, nestedFields: string[] = []): PartialGraphQLInfo => {
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
            if (!isObject(nestedValue)) break;
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
 * Removes the "__typename" field recursively from a JSON-serializable object
 * @param obj - JSON-serializable object with possible __typename fields
 * @return obj without __typename fields
 */
const removeTypenames = (obj: { [x: string]: any }): { [x: string]: any } => {
    return JSON.parse(JSON.stringify(obj, (k, v) => (k === '__typename') ? undefined : v))
}

/**
 * Converts shapes 1 and 2 in a GraphQL to Prisma conversion to shape 2
 * @param info - GraphQL info object, or result of this function
 * @param relationshipMap - Map of relationship names to typenames
 * @returns Partial Prisma select. This can be passed into the function again without changing the result.
 */
export const toPartialGraphQLInfo = (info: GraphQLInfo | PartialGraphQLInfo, relationshipMap: RelationshipMap): PartialGraphQLInfo | undefined => {
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
    // If fields are in the shape of a comment thread search query, convert to a Prisma select object
    else if (select.hasOwnProperty('endCursor') && select.hasOwnProperty('totalThreads') && select.hasOwnProperty('threads')) {
        select = select.threads.comment
    }
    // Inject __typename fields
    select = injectTypenames(select, relationshipMap);
    return select;
}

/**
 * Converts shapes 2 and 3 of a GraphQL to Prisma conversion to shape 3. 
 * This function is useful when we want to check the shape of the requested data, 
 * but not actually query the database.
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Prisma select object with calculated fields, unions and join tables removed, 
 * and count fields and __typenames added,
 */
export const toPartialPrismaSelect = (partial: PartialGraphQLInfo | PartialPrismaSelect): PartialPrismaSelect => {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recursively call selectToDB
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value as PartialGraphQLInfo | PartialPrismaSelect);
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
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export const selectHelper = (partial: PartialGraphQLInfo | PartialPrismaSelect): PrismaSelect | undefined => {
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified: PartialPrismaSelect = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete __typename fields
    modified = removeTypenames(modified);
    // Pad every relationship with "select"
    modified = padSelect(modified);
    return modified;
}

/**
 * Converts shapes 4 of the GraphQL to Prisma conversion to shape 1. Used to format the result of a query.
 * @param data Prisma object
 * @param partialInfo PartialGraphQLInfo object
 * @returns Valid GraphQL object
 */
export function modelToGraphQL<GraphQLModel>(data: { [x: string]: any }, partialInfo: PartialGraphQLInfo): GraphQLModel {
    // Remove top-level union from partialInfo, if necessary
    // If every key starts with a capital letter, it's a union. 
    // There's a catch-22 here which we must account for. Since "data" has not 
    // been shaped yet, it won't match the shape of "partialInfo". But we can't do 
    // this after shaping "data" because we need to know the type of the union. 
    // To account for this, we call modelToGraphQL on each union, to check which one matches "data"
    if (Object.keys(partialInfo).every(k => k[0] === k[0].toUpperCase())) {
        // Find the union type which matches the shape of value. 
        let matchingType: string | undefined;
        for (const unionType of Object.keys(partialInfo)) {
            const unionPartial = partialInfo[unionType];
            if (!isObject(unionPartial)) continue;
            const convertedData = modelToGraphQL(data, unionPartial as any);
            if (subsetsMatch(convertedData, unionPartial)) matchingType = unionType;
        }
        if (matchingType) {
            partialInfo = partialInfo[matchingType] as PartialGraphQLInfo;
        }
    }
    // Convert data to usable shape
    const type: string | undefined = partialInfo?.__typename;
    if (type !== undefined && type in FormatterMap) {
        const formatter: FormatConverter<any> = FormatterMap[type as keyof typeof FormatterMap] as FormatConverter<any>;
        if (formatter.constructUnions) data = formatter.constructUnions(data);
        if (formatter.removeJoinTables) data = formatter.removeJoinTables(data);
        if (formatter.removeCountFields) data = formatter.removeCountFields(data);
    }
    // Then loop through each key/value pair in data and call modelToGraphQL on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partialInfo, check if union
        if (!isObject(partialInfo) || !(key in partialInfo)) continue;
        // If value is an array, call modelToGraphQL on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGraphQL
            data[key] = data[key].map((v: any) => modelToGraphQL(v, partialInfo[key] as PartialGraphQLInfo));
        }
        // If value is an object (and not date), call modelToGraphQL on it
        else if (isRelationshipObject(value)) {
            data[key] = modelToGraphQL(value, (partialInfo as any)[key]);
        }
    }
    return data as any;
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

export interface RelationshipToPrismaArgs<N extends string> {
    data: { [x: string]: any },
    relationshipName: string,
    isAdd: boolean,
    fieldExcludes?: string[],
    relExcludes?: RelationshipTypes[],
    softDelete?: boolean,
    idField?: N,
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
 * @param idField The name of the id field. Defaults to "id"
 */
export const relationshipToPrisma = <N extends string>({
    data,
    relationshipName,
    isAdd,
    fieldExcludes = [],
    relExcludes = [],
    softDelete = false,
    idField = 'id' as N,
}: RelationshipToPrismaArgs<N>): {
    connect?: { [key in N]: string }[],
    disconnect?: { [key in N]: string }[],
    delete?: { [key in N]: string }[],
    create?: { [x: string]: any }[],
    update?: { where: { [key in N]: string }, data: { [x: string]: any } }[],
} => {
    // Determine valid operations, and remove operations that should be excluded
    let ops = isAdd ? [RelationshipTypes.connect, RelationshipTypes.create] : Object.values(RelationshipTypes);
    ops = difference(ops, relExcludes)
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
    if (Array.isArray(converted.connect) && converted.connect.length > 0) converted.connect = converted.connect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.disconnect) && converted.disconnect.length > 0) converted.disconnect = converted.disconnect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.delete) && converted.delete.length > 0) converted.delete = converted.delete.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(converted.update) && converted.update.length > 0) {
        converted.update = converted.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    return converted;
}

export interface JoinRelationshipToPrismaArgs<N extends string> extends RelationshipToPrismaArgs<N> {
    joinFieldName: string, // e.g. organization.tags.tag => 'tag'
    uniqueFieldName: string, // e.g. organization.tags.tag => 'organization_tags_taggedid_tagTag_unique'
    childIdFieldName: string, // e.g. organization.tags.tag => 'tagTag'
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
 * @param idField The name of the id field. Defaults to "id"
 */
export const joinRelationshipToPrisma = <N extends string>({
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
    softDelete = false,
    idField = 'id' as N,
}: JoinRelationshipToPrismaArgs<N>): { [x: string]: any } => {
    let converted: { [x: string]: any } = {};
    // Call relationshipToPrisma to get join data used for one-to-many relationships
    const normalJoinData = relationshipToPrisma({ data, relationshipName, isAdd, fieldExcludes, relExcludes, softDelete, idField })
    // Convert this to support a join table
    if (normalJoinData.hasOwnProperty('connect')) {
        // ex: create: [ { tag: { connect: { id: 'asdf' } } } ] <-- A join table always creates on connects
        for (const id of (normalJoinData?.connect ?? [])) {
            const curr = { [joinFieldName]: { connect: id } };
            converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('disconnect')) {
        // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
        for (const id of (normalJoinData?.disconnect ?? [])) {
            const curr = { [uniqueFieldName]: { [childIdFieldName]: id[idField], [parentIdFieldName]: parentId } };
            converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
        }
    }
    if (normalJoinData.hasOwnProperty('delete')) {
        // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ]
        for (const id of (normalJoinData?.delete ?? [])) {
            const curr = { [uniqueFieldName]: { [childIdFieldName]: id[idField], [parentIdFieldName]: parentId } };
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
        //         where: { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } },
        //         data: { tag: { update: { tag: 'fdas', } } }
        //     }]
        for (const data of (normalJoinData?.update ?? [])) {
            const curr = {
                where: { [uniqueFieldName]: { [childIdFieldName]: data.where[idField], [parentIdFieldName]: parentId } },
                data: { [joinFieldName]: { update: data.data } }
            };
            converted.update = Array.isArray(converted.update) ? [...converted.update, curr] : [curr];
        }
    }
    return converted;
}

/**
 * Converts a GraphQL union into a Prisma select.
 * All possible union fields which the user wants to select are in partialInfo, but they are not separated by type.
 * This function performs a nested intersection between each union type's avaiable fields, and the fields which were 
 * requested by the user.
 * @param partialInfo - partialGraphQLInfo object
 * @param unionField - Name of the union field
 * @param relationshipTupes - array of [relationship type (i.e. how it appears in partialInfo), name to convert the field to]
 * @returns partilPrismaSelect object
 */
export const deconstructUnion = (partialInfo: PartialGraphQLInfo, unionField: string, relationshipTuples: [GraphQLModelType, string][]): any => {
    let { [unionField]: unionData, ...rest } = partialInfo;
    // Create result object
    let converted: { [x: string]: any } = {};
    // If field in partialInfo is not an object, return partialInfo unmodified
    if (!unionData) return partialInfo;
    // Swap keys of union to match their prisma names
    for (const [relType, relName] of relationshipTuples) {
        // If union missing, skip
        if (!isObject(unionData) || !(relType in unionData)) continue;
        const currData = unionData[relType];
        converted[relName] = currData;
    }
    return { ...rest, ...converted };
}

/**
 * Determines if a queried object matches the shape of a GraphQL request object
 * @param obj - queried object
 * @param query - GraphQL request object
 * @returns True if obj matches query
 */
const subsetsMatch = (obj: any, query: any): boolean => {
    // Check that both params are valid objects
    if (obj === null || typeof obj !== 'object' || query === null || typeof query !== 'object') return false;
    // Check if query type is in FormatterMap. 
    // This should hopefully always be the case for the main subsetsMatch call, 
    // but not necessarily for the recursive calls.
    let formattedQuery = query;
    if (query.__typename in FormatterMap) {
        // Remove calculated fields from query, since these will not be in obj
        const formatter = FormatterMap[query.__typename as keyof typeof FormatterMap];
        formattedQuery = formatter?.removeCalculatedFields ? formatter.removeCalculatedFields(query) : query;
    }
    // First, check if obj is a join table. If this is the case, what we want to check 
    // is actually one layer down
    let formttedObj = obj;
    if (Object.keys(obj).length === 1 && isRelationshipObject(obj[Object.keys(obj)[0]])) {
        formttedObj = obj[Object.keys(obj)[0]];
    }
    // If query contains any fields which are not in obj, return false
    for (const key of Object.keys(formattedQuery)) {
        // Ignore __typename
        if (key === '__typename') continue;
        // If key is not in object, return false
        if (!formttedObj.hasOwnProperty(key)) {
            return false;
        }
        // If union, check if any of the union types match formttedObj[key]
        else if (key[0] === key[0].toUpperCase()) {
            const unionTypes = Object.keys(formattedQuery[key]);
            const unionSubsetsMatch = unionTypes.some(unionType => subsetsMatch(formttedObj[key], formattedQuery[key][unionType]));
            if (!unionSubsetsMatch) return false;
        }
        // If formttedObj[key] is array, compare to first element of query[key]
        else if (Array.isArray(formttedObj[key])) {
            // Can't check if array is empty
            if (formttedObj[key].length === 0) continue;
            const firstElem = formttedObj[key][0];
            const matches = subsetsMatch(firstElem, formattedQuery[key]);
            if (!matches) return false;
        }
        // If formttedObj[key] is formttedObject, recurse
        else if (isRelationshipObject((formttedObj)[key])) {
            const matches = subsetsMatch(formttedObj[key], formattedQuery[key]);
            if (!matches) return false;
        }
    }
    return true;
}

/**
 * Combines fields from a Prisma object with arbitrarily nested relationships
 * @param data GraphQL-shaped data, where each object contains at least an ID
 * @param partialInfo PartialGraphQLInfo object
 * @returns [objectDict, selectFieldsDict], where objectDict is a dictionary of object arrays (sorted by type), 
 * and selectFieldsDict is a dictionary of select fields for each type (unioned if they show up in multiple places)
 */
const groupIdsByType = (data: { [x: string]: any }, partialInfo: PartialGraphQLInfo): [{ [x: string]: string[] }, { [x: string]: any }] => {
    if (!data || !partialInfo) return [{}, {}];
    let objectIdsDict: { [x: string]: string[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: { [x: string]: any } } } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        let childPartialInfo: PartialGraphQLInfo = partialInfo[key] as any;
        if (childPartialInfo)
            // If every key in childPartialInfo starts with a capital letter, then it is a union.
            // In this case, we must determine which union to use based on the shape of value
            if (isObject(childPartialInfo) && Object.keys(childPartialInfo).every(k => k[0] === k[0].toUpperCase())) {
                // Find the union type which matches the shape of value
                let matchingType: string | undefined;
                for (const unionType of Object.keys(childPartialInfo)) {
                    if (subsetsMatch(value, childPartialInfo[unionType])) matchingType = unionType;
                }
                // If no union type matches, skip
                if (!matchingType) continue;
                // If union type, update child partial
                childPartialInfo = childPartialInfo[matchingType] as PartialGraphQLInfo;
            }
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through groupSupplementsByType
            for (const v of value) {
                const [childObjectsDict, childSelectFieldsDict] = groupIdsByType(v, childPartialInfo);
                for (const [childType, childObjects] of Object.entries(childObjectsDict)) {
                    objectIdsDict[childType] = objectIdsDict[childType] ?? [];
                    objectIdsDict[childType].push(...childObjects);
                }
                selectFieldsDict = merge(selectFieldsDict, childSelectFieldsDict);
            }
        }
        // If value is an object (and not date), add it to the correct key in objectDict
        else if (isRelationshipObject(value)) {
            // Pass value through groupIdsByType
            const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(value, childPartialInfo);
            for (const [childType, childObjects] of Object.entries(childObjectIdsDict)) {
                objectIdsDict[childType] = objectIdsDict[childType] ?? [];
                objectIdsDict[childType].push(...childObjects);
            }
            selectFieldsDict = merge(selectFieldsDict, childSelectFieldsDict);
        }
        else if (key === 'id' && partialInfo.__typename) {
            // Add to objectIdsDict
            const type: string = partialInfo.__typename;
            objectIdsDict[type] = objectIdsDict[type] ?? [];
            objectIdsDict[type].push(value);
        }
    }
    // Add keys to selectFieldsDict
    const currType = partialInfo?.__typename;
    if (currType) {
        selectFieldsDict[currType] = merge(selectFieldsDict[currType] ?? {}, partialInfo);
    }
    // Return objectDict and selectFieldsDict
    return [objectIdsDict, selectFieldsDict];
}

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
    return merge(result, objectsById[data.id])
}

// TODO might not work if ID appears multiple times in data, where the first
// result is not the one we want
/**
 * Picks an object from a nested object, using the given ID
 * @param data Object array to pick from
 * @param id ID to pick
 * @returns Requested object with all its fields and children included. If object not found, 
 * returns { id }
 */
const pickObjectById = (data: any, id: string): { [x: string]: any } => {
    // Stringify data, so we can perform search of ID
    const dataString = JSON.stringify(data);
    // Find the location in the string where the ID is. 
    // Data is only found if there are more fields than just the ID
    const searchString = `"id":"${id}",`;
    const idIndex = dataString.indexOf(searchString);
    // If ID not found
    if (idIndex === -1) return { id };
    // Loop backwards until we find the start of the object (i.e. first unmatched open bracket before ID)
    let openBracketCounter = 0;
    let inQuotes = false;
    let startIndex = idIndex - 1;
    let lastChar = dataString[idIndex];
    while (startIndex >= 0) {
        const currChar = dataString[startIndex];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // Don't count bracket if it appears in quotes (i.e. part of a string)
            if (!inQuotes) {
                if (dataString[startIndex] === '{') openBracketCounter++;
                else if (dataString[startIndex] === '}') openBracketCounter--;
                // If we found the closing bracket, we're done
                if (openBracketCounter === 1) {
                    break;
                }
            }
            else if (dataString[startIndex] === '"') inQuotes = !inQuotes;
        } else startIndex--;
        lastChar = dataString[startIndex];
        startIndex--;
    }
    // If start is not found
    if (startIndex === -1) return { id };
    // Loop forwards through string until we find the end of the object
    openBracketCounter = 1;
    inQuotes = false;
    let endIndex = idIndex + searchString.length;
    lastChar = dataString[idIndex + searchString.length];
    while (endIndex < dataString.length) {
        const currChar = dataString[endIndex];
        // If current and last char are "\", then the next character is escaped and should be ignored
        if (currChar !== '\\' && lastChar !== '\\') {
            // Don't count bracket if it appears in quotes (i.e. part of a string)
            if (!inQuotes) {
                if (dataString[endIndex] === '{') openBracketCounter++;
                else if (dataString[endIndex] === '}') openBracketCounter--;
                // If we found the closing bracket, we're done
                if (openBracketCounter === 0) {
                    break;
                }
            }
            else if (dataString[endIndex] === '"') inQuotes = !inQuotes;
        } else endIndex++;
        lastChar = dataString[endIndex];
        endIndex++;
    }
    // If end is not found, return undefined
    if (endIndex === dataString.length) return { id };
    // Return object
    return JSON.parse(dataString.substring(startIndex, endIndex + 1));
}

/**
 * Adds supplemental fields to the select object, and all of its relationships (and their relationships, etc.)
 * Groups objects types together, so database is called only once for each type.
 * @param prisma Prisma client
 * @param userId Requesting user's ID
 * @param data Array of GraphQL-shaped data, where each object contains at least an ID
 * @param partialInfo PartialGraphQLInfo object
 * @returns data array with supplemental fields added to each object
 */
export const addSupplementalFields = async (
    prisma: PrismaType,
    userId: string | null,
    data: ({ [x: string]: any } | null | undefined)[],
    partialInfo: PartialGraphQLInfo | PartialGraphQLInfo[],
): Promise<{ [x: string]: any }[]> => {
    // Group data IDs and select fields by type. This is needed to reduce the number of times 
    // the database is called, as we can query all objects of the same type at once
    let objectIdsDict: { [x: string]: string[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: any } } = {};
    for (let i = 0; i < data.length; i++) {
        const currData = data[i];
        const currPartialInfo = Array.isArray(partialInfo) ? partialInfo[i] : partialInfo;
        if (!currData || !currPartialInfo) continue;
        const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(currData, currPartialInfo);
        // Merge each array in childObjectIdsDict into objectIdsDict
        for (const [childType, childObjects] of Object.entries(childObjectIdsDict)) {
            objectIdsDict[childType] = objectIdsDict[childType] ?? [];
            objectIdsDict[childType].push(...childObjects);
        }
        // Merge each object in childSelectFieldsDict into selectFieldsDict
        selectFieldsDict = merge(selectFieldsDict, childSelectFieldsDict);
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
 * @param partials Array of PartialGraphQLInfo objects, in the same order as data arrays
 * @param keys Keys to associate with each data array
 * @param userId Requesting user's ID
 * @param prisma Prisma client
 * @returns Object with keys equal to objectTypes, and values equal to arrays of objects with supplemental fields added
 */
export const addSupplementalFieldsMultiTypes = async (
    data: { [x: string]: any }[][],
    partialInfos: PartialGraphQLInfo[],
    keys: string[],
    userId: string | null,
    prisma: PrismaType,
): Promise<{ [x: string]: any[] }> => {
    // Flatten data array
    const combinedData = flatten(data);
    // Create an array of partials, that match the data array
    let combinedPartialInfo: PartialGraphQLInfo[] = [];
    for (let i = 0; i < data.length; i++) {
        const currPartial = partialInfos[i];
        // Push partial for each data array
        for (let j = 0; j < data[i].length; j++) {
            combinedPartialInfo.push(currPartial);
        }
    }
    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(prisma, userId, combinedData, combinedPartialInfo);
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
 * Helper function for reading one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL find by ID input object
 * @param info GraphQL info object
 * @param model Business layer object
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel>(
    userId: string | null,
    input: FindByIdOrHandleInput,
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<GraphQLModel, any>,
): Promise<RecursivePartial<GraphQLModel>> {
    // Validate input
    if (!input.id && !input.handle)
        throw new CustomError(CODE.InvalidArgs, 'id is required', { code: genErrorCode('0019') });
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, model.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0020') });
    // Uses __typename to determine which Prisma object is being queried
    const objectType: string | undefined = partialInfo.__typename;
    if (objectType === undefined || !(objectType in PrismaMap)) {
        throw new CustomError(CODE.InternalError, `${objectType} missing in PrismaMap`, { code: genErrorCode('0021') });
    }
    // Get the Prisma object
    const prismaObject = (PrismaMap[objectType as keyof typeof PrismaMap] as any)(model.prisma);
    let object = input.id ?
        await prismaObject.findUnique({ where: { id: input.id }, ...selectHelper(partialInfo) }) :
        await prismaObject.findFirst({ where: { handle: input.handle }, ...selectHelper(partialInfo) });
    if (!object)
        throw new CustomError(CODE.NotFound, `${objectType} not found`, { code: genErrorCode('0022') });
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, partialInfo) as RecursivePartial<GraphQLModel>;
    // If logged in and object has view count, handle it
    if (userId && objectType in ViewFor) {
        ViewModel(model.prisma).view(userId, { forId: object.id, title: '', viewFor: objectType as any }); //TODO add title, which requires user's language
    }
    return (await addSupplementalFields(model.prisma, userId, [formatted], partialInfo))[0] as RecursivePartial<GraphQLModel>;
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
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<GraphQLModel, SearchInput>,
    additionalQueries?: { [x: string]: any },
    addSupplemental = true,
): Promise<PaginatedSearchResult> {
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    // Uses __typename to determine which Prisma object is being queried
    const objectType: string | undefined = partialInfo.__typename;
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
        ...selectHelper(partialInfo)
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
    formattedNodes = formattedNodes.map(n => modelToGraphQL(n, partialInfo as PartialGraphQLInfo));
    formattedNodes = await addSupplementalFields(model.prisma, userId, formattedNodes, partialInfo);
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
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<GraphQLModel, any>,
): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0025') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support create', { code: genErrorCode('0026') });
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, model.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0027') });
    const cudResult = await model.cud({ partialInfo, userId, createMany: [input] });
    const { created } = cudResult;
    if (created && created.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partialInfo.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = created.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Create,
                object1Type: objectType,
                object1Id: c.id,
            }));
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Create" log', { code: genErrorCode('0194'), error }));
        }
        return (await addSupplementalFields(model.prisma, userId, created, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in createHelper', { code: genErrorCode('0028') });
}

/**
 * Helper function for updating one object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL update input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @param where Function to create where clause for update (defaults to (obj) => ({ id: obj.id }))
 * @returns GraphQL response object
 */
export async function updateHelper<GraphQLModel>(
    userId: string | null,
    input: any,
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<GraphQLModel, any>,
    where: (obj: any) => { [x: string]: any } = (obj) => ({ id: obj.id }),
): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0029') });
    if (!model.cud)
        throw new CustomError(CODE.InternalError, 'Model does not support update', { code: genErrorCode('0030') });
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0031') });
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: where(input), data: input };
    const { updated } = await model.cud({ partialInfo, userId, updateMany: [shapedInput] });
    if (updated && updated.length > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = partialInfo.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            const logs = updated.map((c: any) => ({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Update,
                object1Type: objectType,
                object1Id: c.id,
            }));
            // No need to await this, since it is not needed for the response
            Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Update" log', { code: genErrorCode('0195'), error }));
        }
        return (await addSupplementalFields(model.prisma, userId, updated, partialInfo))[0] as any;
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
    const { deleted } = await model.cud({ partialInfo: {}, userId, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = model.relationshipMap.__typename;
        if (objectType === 'Organization' || objectType === 'Project' || objectType === 'Routine' || objectType === 'Standard') {
            // No need to await this, since it is not needed for the response
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: LogType.Delete,
                object1Type: objectType,
                object1Id: input.id,
            }).catch(error => logger.log(LogLevel.error, 'Failed creating "Delete" log', { code: genErrorCode('0196'), error }));
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
    const { deleted } = await model.cud({ partialInfo: {}, userId, deleteMany: input.ids });
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
        // No need to await this, since it is not needed for the response
        Log.collection.insertMany(logs).catch(error => logger.log(LogLevel.error, 'Failed creating "Delete" logs', { code: genErrorCode('0197'), error }));
    }
    return deleted
}

/**
 * Helper function for copying an object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @returns GraphQL Success response object
 */
export async function copyHelper(
    userId: string | null,
    input: CopyInput,
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<any, any>,
): Promise<any> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to copy object', { code: genErrorCode('0229') });
    if (!model.duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support copy', { code: genErrorCode('0230') });
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, ({
        '__typename': GraphQLModelType.Copy,
        'node': GraphQLModelType.Node,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
        'routine': GraphQLModelType.Routine,
        'standard': GraphQLModelType.Standard,
    }));
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0231') });
    const { object } = await model.duplicate({ userId, objectId: input.id, isFork: false, createCount: 0 });
    // If not a node, log for stats
    if (input.objectType !== CopyType.Node) {
        // No need to await this, since it is not needed for the response
        Log.collection.insertOne({
            timestamp: Date.now(),
            userId: userId,
            action: LogType.Copy,
            object1Type: input.objectType,
            object1Id: input.id,
        }).catch(error => logger.log(LogLevel.error, 'Failed creating "Copy" log', { code: genErrorCode('0232'), error }));
    }
    const fullObject = await readOneHelper(userId, { id: object.id }, (partialInfo as any)[input.objectType.toLowerCase()], model);
    return fullObject;
}

/**
 * Helper function for forking an object in a single line
 * @param userId ID of user making the request
 * @param input GraphQL delete one input object
 * @param info GraphQL info object
 * @param model Business layer model
 * @returns GraphQL Success response object
 */
export async function forkHelper(
    userId: string | null,
    input: ForkInput,
    info: GraphQLInfo | PartialGraphQLInfo,
    model: ModelBusinessLayer<any, any>,
): Promise<any> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to fork object', { code: genErrorCode('0233') });
    if (!model.duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support fork', { code: genErrorCode('0234') });
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, ({
        '__typename': GraphQLModelType.Fork,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
        'routine': GraphQLModelType.Routine,
        'standard': GraphQLModelType.Standard,
    }));
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0235') });
    const { object } = await model.duplicate({ userId, objectId: input.id, isFork: false, createCount: 0 });
    // Log for stats
    // No need to await this, since it is not needed for the response
    Log.collection.insertOne({
        timestamp: Date.now(),
        userId: userId,
        action: LogType.Fork,
        object1Type: input.objectType,
        object1Id: input.id,
    }).catch(error => logger.log(LogLevel.error, 'Failed creating "Fork" log', { code: genErrorCode('0236'), error }));
    const fullObject = await readOneHelper(userId, { id: object.id }, (partialInfo as any)[input.objectType.toLowerCase()], model);
    return fullObject;
}