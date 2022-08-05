// Components for providing basic functionality to model objects
import { CopyInput, CopyType, Count, DeleteManyInput, DeleteOneInput, ForkInput, PageInfo, Success, TimeFrame } from '../../schema/types';
import { PrismaType, RecursivePartial } from '../../types';
import { GraphQLResolveInfo } from 'graphql';
import { CommentModel } from './comment';
import { NodeModel } from './node';
import { OrganizationModel } from './organization';
import { ProjectModel } from './project';
import { ReportModel } from './report';
import { ResourceModel } from './resource';
import { RoleModel } from './role';
import { RoutineModel } from './routine';
import { StandardModel } from './standard';
import { TagModel } from './tag';
import { UserModel } from './user';
import { StarModel } from './star';
import { VoteModel } from './vote';
import { EmailModel } from './email';
import { CustomError } from '../../error';
import { CODE, isObject, ViewFor } from '@local/shared';
import { ProfileModel } from './profile';
import { MemberModel } from './member';
import { resolveGraphQLInfo } from '../../utils';
import { InputItemModel } from './inputItem';
import { OutputItemModel } from './outputItem';
import { ResourceListModel } from './resourceList';
import { TagHiddenModel } from './tagHidden';
import { Log, LogType } from '../../models/nosql';
import { genErrorCode, logger, LogLevel } from '../../logger';
import { ViewModel } from './view';
import { RunModel } from './run';
import pkg from 'lodash';
import { WalletModel } from './wallet';
import { RunStepModel } from './runStep';
import { NodeRoutineListModel } from './nodeRoutineList';
import { RunInputModel } from './runInput';
import { ValueOf } from '@local/shared';
import { validate as uuidValidate } from 'uuid';
const { difference, flatten, merge } = pkg;


//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

export const GraphQLModelType = {
    Comment: 'Comment',
    Copy: 'Copy',
    DevelopPageResult: 'DevelopPageResult',
    Email: 'Email',
    Fork: 'Fork',
    Handle: 'Handle',
    HistoryPageResult: 'HistoryPageResult',
    HomePageResult: 'HomePageResult',
    InputItem: 'InputItem',
    LearnPageResult: 'LearnPageResult',
    Member: 'Member',
    Node: 'Node',
    NodeEnd: 'NodeEnd',
    NodeLoop: 'NodeLoop',
    NodeRoutineList: 'NodeRoutineList',
    NodeRoutineListItem: 'NodeRoutineListItem',
    Organization: 'Organization',
    OutputItem: 'OutputItem',
    Profile: 'Profile',
    Project: 'Project',
    Report: 'Report',
    ResearchPageResult: 'ResearchPageResult',
    Resource: 'Resource',
    ResourceList: 'ResourceList',
    Role: 'Role',
    Routine: 'Routine',
    Run: 'Run',
    RunInput: 'RunInput',
    RunStep: 'RunStep',
    Standard: 'Standard',
    Star: 'Star',
    Tag: 'Tag',
    TagHidden: 'TagHidden',
    User: 'User',
    View: 'View',
    Vote: 'Vote',
    Wallet: 'Wallet',
} as const
export type GraphQLModelType = ValueOf<typeof GraphQLModelType>;

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
    permissions?: (prisma: PrismaType) => Permissioner<PermissionObject, SearchInput>;
    verify?: { [x: string]: any };
    query?: (prisma: PrismaType) => Querier;
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
export type FormatConverter<GraphQLModel, PermissionObject> = {
    /**
     * Maps relationship names to their GraphQL type. 
     * If the relationship is a union (i.e. has mutliple possible types), 
     * the GraphQL type will be an object of field/GraphQLModelType pairs.
     */
    relationshipMap: RelationshipMap;
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
    customQueries?: (input: SearchInput) => { [x: string]: any };
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
    get({ objects, permissions, userId }: {
        objects: ({ id: string } & { [x: string]: any })[],
        permissions?: PermissionObject[] | null,
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
        userId: string | null,
    }): Promise<'full' | 'public' | 'none'>
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

/**
 * Maps model types to various helper functions
 */
export const ObjectMap: { [key in GraphQLModelType]?: ModelLogic<any, any, any> } = {
    'Comment': CommentModel,
    'Email': EmailModel,
    'InputItem': InputItemModel,
    'Member': MemberModel, // TODO create searcher for members
    'Node': NodeModel,
    'NodeRoutineList': NodeRoutineListModel,
    'Organization': OrganizationModel,
    'OutputItem': OutputItemModel,
    'Profile': ProfileModel,
    'Project': ProjectModel,
    'Report': ReportModel,
    'Resource': ResourceModel,
    'ResourceList': ResourceListModel,
    'Role': RoleModel,
    'Routine': RoutineModel,
    'Run': RunModel,
    'RunInput': RunInputModel,
    'Standard': StandardModel,
    'RunStep': RunStepModel,
    'Star': StarModel,
    'Tag': TagModel,
    'TagHidden': TagHiddenModel,
    'User': UserModel,
    'Vote': VoteModel,
    'View': ViewModel,
    'Wallet': WalletModel,
}

/**
 * Determines if an object is a relationship object, and not an array of relationship objects.
 * @param obj - object to check
 * @returns True if obj is a relationship object, false otherwise
 */
const isRelationshipObject = (obj: any): boolean => isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]';

/**
 * Determines if an object is an array of relationship objects, and not a relationship object.
 * @param obj - object to check
 * @returns True if obj is an array of relationship objects, false otherwise
 */
const isRelationshipArray = (obj: any): boolean => Array.isArray(obj) && obj.every(e => isRelationshipObject(e));

/**
 * Filters out any invalid IDs from an array of IDs.
 * @param ids - array of IDs to filter
 * @returns Array of valid IDs
 */
const onlyValidIds = (ids: string[]): string[] => ids.filter(id => uuidValidate(id));

/**
 * Lowercases the first letter of a string
 * @param str String to lowercase
 * @returns Lowercased string
 */
export function lowercaseFirstLetter(str: string): string {
    return str.charAt(0).toLowerCase() + str.slice(1);
}

/**
 * Uppercases the first letter of a string
 * @param str String to capitalize
 * @returns Uppercased string
 */
export function uppercaseFirstLetter(str: string): string {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Helper function for adding join tables between 
 * many-to-many relationship parents and children
 * @param partialInfo - GraphQL-shaped object
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
        ['User', 'createdByUser'],
        ['Organization', 'createdByOrganization']
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
        ['User', 'user'],
        ['Organization', 'organization']
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
        if (nestedValue !== undefined && typeof nestedValue !== 'object') {
            relationshipMap = ObjectMap[nestedValue]?.format?.relationshipMap;
        }
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
    const formatter: FormatConverter<any, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (formatter) {
        if (formatter.removeSupplementalFields) result = formatter.removeSupplementalFields(result);
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
    const formatter: FormatConverter<GraphQLModel, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (formatter) {
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
 * @param relationshipTuples - array of [relationship type (i.e. how it appears in partialInfo), name to convert the field to]
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
    console.log('subsets match start', JSON.stringify(obj), '\n', JSON.stringify(query), '\n\n');
    // Check that both params are valid objects
    if (obj === null || typeof obj !== 'object' || query === null || typeof query !== 'object') return false;
    // Check if query type is in FormatterMap. 
    // This should hopefully always be the case for the main subsetsMatch call, 
    // but not necessarily for the recursive calls.
    let formattedQuery = query;
    const formatter: FormatConverter<any, any> | undefined = typeof query?.__typename === 'string' ? ObjectMap[query.__typename as keyof typeof ObjectMap]?.format : undefined;
    if (formatter) {
        // Remove calculated fields from query, since these will not be in obj
        formattedQuery = formatter?.removeSupplementalFields ? formatter.removeSupplementalFields(query) : query;
    } else console.log('no subste formatter', JSON.stringify(query), '\n\n');
    // First, check if obj is a join table. If this is the case, what we want to check 
    // is actually one layer down
    let formattedObj = obj;
    if (Object.keys(obj).length === 1 && isRelationshipObject(obj[Object.keys(obj)[0]])) {
        formattedObj = obj[Object.keys(obj)[0]];
    }
    // If query contains any fields which are not in obj, return false
    for (const key of Object.keys(formattedQuery)) {
        // Ignore __typename
        if (key === '__typename') continue;
        // If union, check if any of the union types match formattedObj
        if (key[0] === key[0].toUpperCase()) {
            const unionTypes = Object.keys(formattedQuery);
            const unionSubsetsMatch = unionTypes.some(unionType => subsetsMatch(formattedObj, formattedQuery[unionType]));
            if (!unionSubsetsMatch) return false;
        }
        // If key is not in object, return false
        else if (!formattedObj.hasOwnProperty(key)) {
            return false;
        }
        // If formattedObj[key] is array, compare to first element of query[key]
        else if (Array.isArray(formattedObj[key])) {
            // Can't check if array is empty
            if (formattedObj[key].length === 0) continue;
            const firstElem = formattedObj[key][0];
            const matches = subsetsMatch(firstElem, formattedQuery[key]);
            if (!matches) return false;
        }
        // If formattedObj[key] is formattedObject, recurse
        else if (isRelationshipObject((formattedObj)[key])) {
            const matches = subsetsMatch(formattedObj[key], formattedQuery[key]);
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
    console.log('groupIdsByType', JSON.stringify(data), '\n', JSON.stringify(partialInfo), '\n\n');
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
const pickObjectById = (data: any, id: string): ({ id: string } & { [x: string]: any }) => {
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
    console.log('add supp start', JSON.stringify(data), '\n\n');
    if (data.length === 0) return [];
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
    console.log('objectIdsDicct', JSON.stringify(objectIdsDict));

    // Loop through each type in objectIdsDict
    for (const [type, ids] of Object.entries(objectIdsDict)) {
        // Find the data for each id in ids. Since the data parameter is an array,
        // we must loop through each element in it and call pickObjectById
        const objectData = ids.map((id: string) => pickObjectById(data, id));
        // Now that we have the data for each object, we can add the supplemental fields
        const formatter: FormatConverter<any, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
        const valuesWithSupplements = formatter?.addSupplementalFields ?
            await formatter.addSupplementalFields({ objects: objectData, partial: selectFieldsDict[type], prisma, userId }) :
            objectData;
        // Add each value to objectsById
        for (const v of valuesWithSupplements) {
            objectsById[v.id] = v;
        }
    }
    // Convert objectsById dictionary back into shape of data
    let result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, objectsById));
    console.log('add supp end', JSON.stringify(result), '\n\n');
    return result
}

/**
 * Combines addSupplementalFields calls for multiple object types
 * @param data Array of arrays, where each array is a list of the same object type queried from the database
 * @param partialInfos Array of PartialGraphQLInfo objects, in the same order as data arrays
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
    console.log('multi a');
    // Flatten data array
    const combinedData = flatten(data);
    console.log('multi b');
    // Create an array of partials, that match the data array
    let combinedPartialInfo: PartialGraphQLInfo[] = [];
    for (let i = 0; i < data.length; i++) {
        const currPartial = partialInfos[i];
        // Push partial for each data array
        for (let j = 0; j < data[i].length; j++) {
            combinedPartialInfo.push(currPartial);
        }
    }
    console.log('multi c');
    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(prisma, userId, combinedData, combinedPartialInfo);
    console.log('multi d');
    // Convert combinedResult into object with keys equal to objectTypes, and values equal to arrays of those types
    const formatted: { [y: string]: any[] } = {};
    let start = 0;
    for (let i = 0; i < keys.length; i++) {
        const currKey = keys[i];
        const end = start + data[i].length;
        formatted[currKey] = combinedResult.slice(start, end);
        start = end;
    }
    console.log('multi e');
    return formatted;
}

type PermissionsHelper<PermissionObject> = {
    /**
     * Array of actions to check for
     */
    actions: string[];
    model: ModelLogic<any, any, PermissionObject>;
    object: ({ id: string } & { [x: string]: any });
    prisma: PrismaType;
    userId: string | null;
}

//TODO needs better typescript handling
/**
 * Helper function to verify that a user has the correct permissions to perform an action on an object.
 * @returns True if the user has the correct permissions, false otherwise
 */
export async function permissionsCheck<PermissionObject>({
    actions,
    model,
    object,
    prisma,
    userId,
}: PermissionsHelper<PermissionObject>): Promise<boolean> {
    console.log('permissionscheck a', actions, JSON.stringify(object), '\n\n');
    if (!model.permissions) return true;
    // Query object's permissions
    const perms = await model.permissions(prisma).get({ objects: [object], userId });
    console.log('permissionscheck b', JSON.stringify(perms), '\n\n');
    for (const action of actions) {
        if (!perms[0][action as keyof PermissionObject]) {
            return false;
        }
    }
    return true;
}

type ReadOneHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for reading one object in a single line
 * @returns GraphQL response object
 */
export async function readOneHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    userId,
}: ReadOneHelperProps<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>> {
    const objectType = model.format.relationshipMap.__typename;
    // Validate input
    if (!input.id && !input.handle)
        throw new CustomError(CODE.InvalidArgs, 'id is required', { code: genErrorCode('0019') });
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0020') });
    // Check permissions
    const authorized = await permissionsCheck({
        model,
        object: { id: input.id || input.handle },
        actions: ['canView'],
        prisma,
        userId,
    });
    if (!authorized) throw new CustomError(CODE.Unauthorized, `Not allowed to read object`, { code: genErrorCode('0249') });
    // Get the Prisma object
    let object = input.id ?
        await (model.prismaObject(prisma) as any).findUnique({ where: { id: input.id }, ...selectHelper(partialInfo) }) :
        await (model.prismaObject(prisma) as any).findFirst({ where: { handle: input.handle }, ...selectHelper(partialInfo) });
    if (!object)
        throw new CustomError(CODE.NotFound, `${objectType} not found`, { code: genErrorCode('0022') });
    // Return formatted for GraphQL
    let formatted = modelToGraphQL(object, partialInfo) as RecursivePartial<GraphQLModel>;
    console.log('read one formatted', JSON.stringify(formatted), '\n\n')
    // If logged in and object has view count, handle it
    if (userId && objectType in ViewFor) {
        ViewModel.mutate(prisma).view(userId, { forId: object.id, title: '', viewFor: objectType as any }); //TODO add title, which requires user's language
    }
    return (await addSupplementalFields(prisma, userId, [formatted], partialInfo))[0] as RecursivePartial<GraphQLModel>;
}

type ReadManyHelperProps<GraphQLModel, SearchInput extends SearchInputBase<any>> = {
    additionalQueries?: { [x: string]: any };
    /**
     * Decides if queried data should be called. Defaults to true. 
     * You may want to set this to false if you are calling readManyHelper multiple times, so you can do this 
     * later in one call
     */
    addSupplemental?: boolean;
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, SearchInput, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for reading many objects in a single line.
 * Cursor-based search. Supports pagination, sorting, and filtering by string.
 * @returns Paginated search result
 */
export async function readManyHelper<GraphQLModel, SearchInput extends SearchInputBase<any>>({
    additionalQueries,
    addSupplemental = true,
    info,
    input,
    model,
    prisma,
    userId,
}: ReadManyHelperProps<GraphQLModel, SearchInput>): Promise<PaginatedSearchResult> {
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0023') });
    // Make sure ID is in partialInfo, since this is required for cursor-based search
    partialInfo.id = true;
    console.log('readManyHelper a', JSON.stringify(selectHelper(partialInfo)), '\n', JSON.stringify(input), '\n\n');
    // Check permissions
    // TODO need different permissions than readonehelper. Needs to use requested fields to determine which permissions to check.
    // For example, a routines query with a specified organizationId input must check read permission on the organizationId, 
    // and use that to determine if private routines can be returned.
    // Create query for specified ids
    const idQuery = (Array.isArray(input.ids)) ? ({ id: { in: onlyValidIds(input.ids) } }) : undefined;
    // Determine text search query
    const searchQuery = (input.searchString && model.search?.getSearchStringQuery) ? model.search.getSearchStringQuery(input.searchString) : undefined;
    // Determine createdTimeFrame query
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Determine updatedTimeFrame query
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Create type-specific queries
    let typeQuery = model.search?.customQueries ? model.search.customQueries(input) : undefined;
    // Combine queries
    const where = { ...additionalQueries, ...idQuery, ...searchQuery, ...createdQuery, ...updatedQuery, ...typeQuery };
    // Determine sort order
    const orderBy = model.search?.getSortQuery ? model.search.getSortQuery(input.sortBy ?? model.search.defaultSort) : undefined;
    // Find requested search array
    console.log('getting search results', JSON.stringify({
        where,
        orderBy,
        take: input.take ?? 20,
        skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        cursor: input.after ? {
            id: input.after
        } : undefined,
        ...selectHelper(partialInfo)
    }))
    const searchResults = await (model.prismaObject(prisma) as any).findMany({
        where,
        orderBy,
        take: input.take ?? 20,
        skip: input.after ? 1 : undefined, // First result on cursored requests is the cursor, so skip it
        cursor: input.after ? {
            id: input.after
        } : undefined,
        ...selectHelper(partialInfo)
    });
    console.log('got search results', JSON.stringify(searchResults), '\n\n')
    // If there are results
    let paginatedResults: PaginatedSearchResult;
    if (searchResults.length > 0) {
        // Find cursor
        const cursor = searchResults[searchResults.length - 1].id;
        // Query after the cursor to check if there are more results
        const hasNextPage = await (model.prismaObject(prisma) as any).findMany({
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
    formattedNodes = await addSupplementalFields(prisma, userId, formattedNodes, partialInfo);
    return { pageInfo: paginatedResults.pageInfo, edges: paginatedResults.edges.map(({ node, ...rest }) => ({ node: formattedNodes.shift(), ...rest })) };
}

type CountHelperProps<GraphQLModel, CountInput extends CountInputBase> = {
    input: CountInput;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    where?: { [x: string]: any };
}

/**
 * Counts the number of objects in the database, optionally filtered by a where clauses
 * @returns The number of matching objects
 */
export async function countHelper<GraphQLModel, CountInput extends CountInputBase>({
    input,
    model,
    prisma,
    where,
}: CountHelperProps<GraphQLModel, CountInput>): Promise<number> {
    // Check permissions
    //TODO use same permissions as readmanyhelper, to determine if private should be counted
    // Create query for created metric
    const createdQuery = timeFrameToPrisma('created_at', input.createdTimeFrame);
    // Create query for created metric
    const updatedQuery = timeFrameToPrisma('updated_at', input.updatedTimeFrame);
    // Count objects that match queries
    return await (model.prismaObject(prisma) as any).count({
        where: {
            ...where,
            ...createdQuery,
            ...updatedQuery,
        },
    });
}

type CreateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for creating one object in a single line.
 * Throws error if not successful.
 * @returns GraphQL response object
 */
export async function createHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    userId,
}: CreateHelperProps<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0025') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support create', { code: genErrorCode('0026') });
    // Partially convert info type
    const partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0027') });
    // Check permissions
    // TODO will need custom permissions. Needs to use input fields to determine which permissions to check.
    // e.g. A routine with a projectId must verify that the user has permission to create a routine in the project (i.e. owns project and hasn't reached max routines inside).
    // Then it also needs to check that the user hasn't reached a max number of overall routines.
    const cudResult = await model.mutate!(prisma).cud!({ partialInfo, userId, createMany: [input] });
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
        return (await addSupplementalFields(prisma, userId, created, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in createHelper', { code: genErrorCode('0028') });
}

type UpdateHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: any;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    userId: string | null;
    where?: (obj: any) => { [x: string]: any };
}

/**
 * Helper function for updating one object in a single line
 * @returns GraphQL response object
 */
export async function updateHelper<GraphQLModel>({
    info,
    input,
    model,
    prisma,
    userId,
    where = (obj) => ({ id: obj.id }),
}: UpdateHelperProps<any>): Promise<RecursivePartial<GraphQLModel>> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to create object', { code: genErrorCode('0029') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support update', { code: genErrorCode('0030') });
    // Partially convert info type
    let partialInfo = toPartialGraphQLInfo(info, model.format.relationshipMap);
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0031') });
    // Check permissions
    //TODO use same permissions as readonehelper (but for updating obviously), but also similar custom
    // logic to createHelper. For example, if a routine is being moved to another project, then need to check 
    // permissions for current and new project.
    // Shape update input to match prisma update shape (i.e. "where" and "data" fields)
    const shapedInput = { where: where(input), data: input };
    const { updated } = await model.mutate!(prisma).cud!({ partialInfo, userId, updateMany: [shapedInput] });
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
        return (await addSupplementalFields(prisma, userId, updated, partialInfo))[0] as any;
    }
    throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in updateHelper', { code: genErrorCode('0032') });
}

type DeleteOneHelperProps = {
    input: DeleteOneInput;
    model: ModelLogic<any, any, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for deleting one object in a single line
 * @returns GraphQL Success response object
 */
export async function deleteOneHelper({
    input,
    model,
    prisma,
    userId,
}: DeleteOneHelperProps): Promise<Success> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete object', { code: genErrorCode('0033') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0034') });
    // Check permissions
    // TODO
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userId, deleteMany: [input.id] });
    if (deleted?.count && deleted.count > 0) {
        // If organization, project, routine, or standard, log for stats
        const objectType = model.format.relationshipMap.__typename;
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

type DeleteManyHelperProps = {
    input: DeleteManyInput;
    model: ModelLogic<any, any, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for deleting many of the same object in a single line
 * @returns GraphQL Count response object
 */
export async function deleteManyHelper({
    input,
    model,
    prisma,
    userId,
}: DeleteManyHelperProps): Promise<Count> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete objects', { code: genErrorCode('0035') });
    if (!model.mutate || !model.mutate(prisma).cud)
        throw new CustomError(CODE.InternalError, 'Model does not support delete', { code: genErrorCode('0036') });
    // Check permissions
    //TODO
    const { deleted } = await model.mutate!(prisma).cud!({ partialInfo: {}, userId, deleteMany: input.ids });
    if (!deleted)
        throw new CustomError(CODE.ErrorUnknown, 'Unknown error occurred in deleteManyHelper', { code: genErrorCode('0037') });
    // If organization, project, routine, or standard, log for stats
    const objectType = model.format.relationshipMap.__typename;
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

type CopyHelperProps<GraphQLModel> = {
    info: GraphQLInfo | PartialGraphQLInfo;
    input: CopyInput;
    model: ModelLogic<GraphQLModel, any, any>;
    prisma: PrismaType;
    userId: string | null;
}

/**
 * Helper function for copying an object in a single line
 * @returns GraphQL Success response object
 */
export async function copyHelper({
    info,
    input,
    model,
    prisma,
    userId,
}: CopyHelperProps<any>): Promise<any> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to copy object', { code: genErrorCode('0229') });
    const duplicate = model.mutate ? model.mutate(prisma).duplicate : undefined;
    if (!duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support copy', { code: genErrorCode('0230') });
    // Check permissions
    //TODO
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, ({
        '__typename': 'Copy',
        'node': 'Node',
        'organization': 'Organization',
        'project': 'Project',
        'routine': 'Routine',
        'standard': 'Standard',
    }));
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0231') });
    const { object } = await duplicate({ userId, objectId: input.id, isFork: false, createCount: 0 });
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
    const fullObject = await readOneHelper({
        info: (partialInfo as any)[lowercaseFirstLetter(input.objectType)],
        input: { id: object.id },
        model,
        prisma,
        userId,
    })
    return fullObject;
}

type ForkHelperProps<GraphQLModelType> = {
    info: GraphQLInfo | PartialGraphQLInfo,
    input: ForkInput,
    model: ModelLogic<GraphQLModelType, any, any>,
    prisma: PrismaType,
    userId: string | null,
}

/**
 * Helper function for forking an object in a single line
 * @returns GraphQL Success response object
 */
export async function forkHelper({
    info,
    input,
    model,
    prisma,
    userId,
}: ForkHelperProps<any>): Promise<any> {
    if (!userId)
        throw new CustomError(CODE.Unauthorized, 'Must be logged in to fork object', { code: genErrorCode('0233') });
    const duplicate = model.mutate ? model.mutate(prisma).duplicate : undefined;
    if (!duplicate)
        throw new CustomError(CODE.InternalError, 'Model does not support fork', { code: genErrorCode('0234') });
    // Check permissions
    //TODO
    // Partially convert info
    let partialInfo = toPartialGraphQLInfo(info, ({
        '__typename': 'Fork',
        'organization': 'Organization',
        'project': 'Project',
        'routine': 'Routine',
        'standard': 'Standard',
    }));
    if (!partialInfo)
        throw new CustomError(CODE.InternalError, 'Could not convert info to partial select', { code: genErrorCode('0235') });
    const { object } = await duplicate({ userId, objectId: input.id, isFork: false, createCount: 0 });
    // Log for stats
    // No need to await this, since it is not needed for the response
    Log.collection.insertOne({
        timestamp: Date.now(),
        userId: userId,
        action: LogType.Fork,
        object1Type: input.objectType,
        object1Id: input.id,
    }).catch(error => logger.log(LogLevel.error, 'Failed creating "Fork" log', { code: genErrorCode('0236'), error }));
    const fullObject = await readOneHelper({
        info: (partialInfo as any)[lowercaseFirstLetter(input.objectType)],
        input: { id: object.id },
        model,
        prisma,
        userId,
    })
    return fullObject;
}

/**
 * Helper function for reading many objects and converting them to a GraphQL response
 * (except for supplemental fields). This is useful when querying feeds
 */
export async function readManyAsFeed<GraphQLModel, SearchInput extends SearchInputBase<any>>({
    additionalQueries,
    info,
    input,
    model,
    prisma,
    userId,
}: Omit<ReadManyHelperProps<GraphQLModel, SearchInput>, 'addSupplemental'>): Promise<any[]> {
    const readManyResult = await readManyHelper({
        additionalQueries,
        addSupplemental: false,
        info,
        input,
        model,
        prisma,
        userId,
    })
    return readManyResult.edges.map(({ node }: any) =>
        modelToGraphQL(node, toPartialGraphQLInfo(info, model.format.relationshipMap) as PartialGraphQLInfo)) as any[]
}

type AddSupplementalFieldsHelper = {
    objects: ({ id: string } & { [x: string]: any })[],
    partial: PartialGraphQLInfo,
    resolvers: [string, (ids: string[]) => Promise<any>][],
}

/**
 * Helper function for simplifying addSupplementalFields
 */
export async function addSupplementalFieldsHelper<GraphQLModel>({
    objects,
    partial,
    resolvers,
}: AddSupplementalFieldsHelper): Promise<RecursivePartial<GraphQLModel>[]> {
    if (!objects || objects.length === 0) return [];
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Get supplemental fields, and inject into objects
    for (const [field, resolver] of resolvers) {
        // If not in partial, skip
        if (!partial[field]) continue;
        console.log('before resolve', field, ids)
        const supplemental = await resolver(ids);
        objects = objects.map((x, i) => ({ ...x, [field]: supplemental[i] }));
        console.log('after resolve', field)
    }
    return objects;
}

type GetSearchStringHelperProps = {
    resolver: ({ insensitive }: { insensitive: { [x: string]: any } }) => { [x: string]: any },
    searchString: string;
}

/**
 * Helper function for getSearchStringQuery
 * @returns GraphQL search query object
 */
export function getSearchStringQueryHelper({
    resolver,
    searchString,
}: GetSearchStringHelperProps): { [x: string]: any } {
    if (searchString.length === 0) return {};
    const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
    return resolver({ insensitive });
}