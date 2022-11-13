// Components for providing basic functionality to model objects
import { PrismaType, RecursivePartial, ReqForUserAuth } from '../types';
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
import { isObject } from '@shared/utils';
import { ProfileModel } from './profile';
import { MemberModel } from './member';
import { resolveGraphQLInfo } from '../utils';
import { InputItemModel } from './inputItem';
import { OutputItemModel } from './outputItem';
import { ResourceListModel } from './resourceList';
import { TagHiddenModel } from './tagHidden';
import { ViewModel } from './view';
import { RunModel } from './run';
import pkg from 'lodash';
import { WalletModel } from './wallet';
import { RunStepModel } from './runStep';
import { NodeRoutineListModel } from './nodeRoutineList';
import { RunInputModel } from './runInput';
import { uuidValidate } from '@shared/uuid';
import { CountMap, FormatConverter, GraphQLInfo, GraphQLModelType, JoinMap, ModelLogic, PartialGraphQLInfo, PartialPrismaSelect, PrismaSelect, RelationshipMap } from './types';
import { TimeFrame, VisibilityType } from '../schema/types';
const { difference, flatten, merge } = pkg;

/**
 * Maps model types to various helper functions
 */
export const ObjectMap: { [key in GraphQLModelType]?: ModelLogic<any, any, any, any> } = {
    // 'Api': ApiModel,
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
export const isRelationshipObject = (obj: any): obj is Object => isObject(obj) && Object.prototype.toString.call(obj) !== '[object Date]';

/**
 * Determines if an object is an array of relationship objects, and not a relationship object.
 * @param obj - object to check
 * @returns True if obj is an array of relationship objects, false otherwise
 */
export const isRelationshipArray = (obj: any): obj is Object[] => Array.isArray(obj) && obj.every(isRelationshipObject);

/**
 * Filters out any invalid IDs from an array of IDs.
 * @param ids - array of IDs to filter
 * @returns Array of valid IDs
 */
export const onlyValidIds = (ids: (string | null | undefined)[]): string[] => ids.filter(id => typeof id === 'string' && uuidValidate(id)) as string[];

/**
 * Filters out any invalid handles from an array of handles.
 * Handles start with a $ and have 3 to 16 characters.
 * @param handles - array of handles to filter
 * @returns Array of valid handles
 */
export const onlyValidHandles = (handles: (string | null | undefined)[]): string[] => handles.filter(handle => typeof handle === 'string' && handle.match(/^\$[a-zA-Z0-9]{3,16}$/)) as string[];

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
 * Idempotent helper function for adding join tables between 
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
        // If the key is in the object, 
        if (partialInfo[key]) {
            // Skip if already padded with join table name
            if (isRelationshipArray(partialInfo[key])) {
                if ((partialInfo[key] as any).every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== 'id')) {
                    result[key] = partialInfo[key];
                    continue;
                }
            } else if (isRelationshipObject(partialInfo[key])) {
                if (Object.keys(partialInfo[key] as any).length === 1 && Object.keys(partialInfo[key] as any)[0] !== 'id') {
                    result[key] = partialInfo[key];
                    continue
                }
            }
            // Otherwise, pad with the join table name
            result[key] = { [value]: partialInfo[key] };
        }
    }
    return {
        ...partialInfo,
        ...result
    }
}

/**
 * Idempotent helper function for removing join tables between
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
                // Check if the join should be applied (i.e. elements are objects with one non-ID key)
                if (obj[key].every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== 'id')) {
                    // Remove the join table from each item in the array
                    result[key] = obj[key].map((item: any) => item[value]);
                }
            } else {
                // Check if the join should be applied (i.e. element is an object with one non-ID key)
                if (isRelationshipObject(obj[key]) && Object.keys(obj[key]).length === 1 && Object.keys(obj[key])[0] !== 'id') {
                    // Otherwise, remove the join table from the object
                    result[key] = obj[key][value];
                }
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
 * Deconstructs a GraphQL object's relationship fields into database fields. It's the opposite of constructRelationshipsHelper
 * @param data - GraphQL-shaped object
 * @param relationshipMap - Mapping of relationship names to their transform shapes
 * @returns DB-shaped object
 */
export const deconstructRelationshipsHelper = <GraphQLModel>(data: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = data;
    // Filter out all fields in the relationshipMap that don't have an object value
    const relationshipFields: [string, { [key: string]: any }][] = Object.entries(relationshipMap).filter(([key, value]) => isRelationshipObject(value)) as any[];
    // For each relationship field
    for (const [key, value] of relationshipFields) {
        // If it's not in data, continue
        if (!data[key]) continue;
        // Get data in union field
        let unionData = data[key];
        // Remove the union field from the result
        delete result[key];
        // If not an object, skip
        if (!isRelationshipObject(unionData)) continue;
        // Determine if data should be wrapped in a "root" field
        const isWrapped = Object.keys(value).length === 1 && Object.keys(value)[0] === 'root';
        const unionMap: { [key: string]: string } = isWrapped ? value.root : value;
        // unionMap is an object where the keys are possible types of the union object, and values are the db field associated with that type
        // Iterate over the possible types
        for (const [type, dbField] of Object.entries(unionMap)) {
            // If the type is in the union data, add the db field to the result. 
            // Don't forget to handle "root" field
            if (unionData[type]) {
                if (isWrapped) {
                    result.root = isRelationshipObject(result.root) ? { ...result.root, [dbField]: unionData[type] } : { [dbField]: unionData[type] };
                } else {
                    result[dbField] = unionData[type];
                }
            }
        }
    }
    return result;
}

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationshipsHelper
 * @param partialInfo - Partial info object
 * @param relationshipMap - Mapping of GraphQL union field names to Prisma object field names
 * @returns partialInfo object with union fields added
 */
export const constructRelationshipsHelper = <GraphQLModel>(partialInfo: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = partialInfo;
    // Filter out all fields in the relationshipMap that don't have an object value
    const relationshipFields: [string, { [key: string]: any }][] = Object.entries(relationshipMap).filter(([key, value]) => isRelationshipObject(value)) as any[];
    // For each relationship field
    for (const [key, value] of relationshipFields) {
        // Determine if data should be unwrapped from a "root" field
        const isWrapped = Object.keys(value).length === 1 && Object.keys(value)[0] === 'root';
        const unionMap: { [key: string]: string } = isWrapped ? value.root : value;
        // For each type, dbField pair
        for (const [_, dbField] of Object.entries(unionMap)) {
            // If the dbField is in the partialInfo
            const isInPartialInfo = isWrapped ? result.root && result.root[dbField] !== undefined : result[dbField] !== undefined;
            if (isInPartialInfo) {
                // Set the union field to the dbField
                if (isWrapped) {
                    result.root = isRelationshipObject(result.root) ? { ...result.root, [key]: result.root[dbField] } : { [key]: result.root[dbField] };
                } else {
                    result[key] = result[dbField];
                }
                // Delete the dbField from the result
                if (isWrapped) {
                    delete result.root[dbField];
                } else {
                    delete result[dbField];
                }
            }
        }
    }
    return result;
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
const injectTypenames = <GraphQLModel>(select: { [x: string]: any }, parentRelationshipMap: RelationshipMap<GraphQLModel>, nestedFields: string[] = []): PartialGraphQLInfo => {
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
        let nestedValue: GraphQLModelType | Partial<RelationshipMap<GraphQLModel>> | undefined = parentRelationshipMap;
        for (const field of nestedFields) {
            if (!isObject(nestedValue)) break;
            if (field in nestedValue) {
                nestedValue = (nestedValue as any)[field];
            }
        }
        if (typeof nestedValue === 'object') nestedValue = nestedValue[selectKey as keyof GraphQLModel] as any;
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
export const toPartialGraphQLInfo = <GraphQLModel>(info: GraphQLInfo | PartialGraphQLInfo, relationshipMap: RelationshipMap<GraphQLModel>): PartialGraphQLInfo | undefined => {
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
 * and count fields and __typenames added
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
        result = deconstructRelationshipsHelper(result, formatter.relationshipMap);
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
    const formatter: FormatConverter<GraphQLModel, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined as any;
    if (formatter) {
        data = constructRelationshipsHelper(data, formatter.relationshipMap);
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
export const timeFrameToPrisma = (fieldName: string, time?: TimeFrame | null | undefined): { [x: string]: any } | undefined => {
    if (!time || (!time.before && !time.after)) return undefined;
    let where: { [x: string]: any } = ({ [fieldName]: {} });
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
 * 
 * Examples when "isOneToOne" is false (the default):
 *  - '123' => [{ id: '123' }]
 *  - { id: '123' } => [{ id: '123' }]
 *  - { name: 'John' } => [{ name: 'John' }]
 *  - ['123', '456'] => [{ id: '123' }, { id: '456' }]
 * 
 * Examples when "isOneToOne" is true:
 * - '123' => { id: '123' }
 * - { id: '123' } => { id: '123' }
 * - { name: 'John' } => { name: 'John' }
 * - ['123', '456'] => { id: '123' }
 * 
 * @param data The data to shape
 * @param excludes The fields to exclude from the shape
 * @param isOneToOne Whether the data is one-to-one (i.e. a single object)
 */
const shapeRelationshipData = (data: any, excludes: string[] = [], isOneToOne: boolean = false): any => {
    const shapeAsMany = (data: any): any => {
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
    // Shape as if "isOneToOne" is fasel
    let result = shapeAsMany(data);
    // Then if "isOneToOne" is true, return the first element
    if (isOneToOne) {
        if (result.length > 0) {
            result = result[0];
        } else {
            result = {};
        }
    }
    return result;
}

export enum RelationshipTypes {
    connect = 'connect',
    disconnect = 'disconnect',
    create = 'create',
    update = 'update',
    delete = 'delete',
}

export interface RelationshipBuilderHelperArgs<IDField extends string, GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }, DBCreate extends { [x: string]: any }, DBUpdate extends { [x: string]: any }> {
    /**
     * The data to convert
     */
    data: { [x: string]: any },
    /**
     * The name of the relationship to convert. This is required because we're grabbing the 
     * relationship from its parent object
     */
    relationshipName: string,
    /**
     * True if we're creating the parent object, instead of updating it. Adding limits 
     * the types of Prisma operations we can perform
     */
    isAdd: boolean,
    /**
     * True if object can be transferred to another parent object. Some cases where this is false are 
     * emails, phone numbers, etc.
     */
    isTransferable?: boolean,
    /**
     * True if relationship is one-to-one
     */
    isOneToOne?: boolean,
    /**
     * Fields to exclude from the relationship data. This can be handled in the shape fields as well,
     * but this is more convenient
     */
    fieldExcludes?: string[],
    /**
     * Prisma operations to exclude from the relationship data. "isAdd" is often sufficient 
     * to determine which operations to exclude, sometimes it's also nice to exluce "connect" and 
     * "disconnect" when we know that a relationship can only be applied to the parent object
     */
    relExcludes?: RelationshipTypes[],
    /**
     * True if object should be soft-deleted instead of hard-deleted. TODO THIS DOES NOT WORK YET
     */
    softDelete?: boolean,
    /**
     * The name of the ID field. Defaults to "id"
     */
    idField?: IDField,
    /**
     * Functions that perform additional formatting on create and update data. This is often 
     * used to shape grandchildren, great-grandchildren, etc.
     */
    shape?: {
        shapeCreate: (userId: string, create: GraphQLCreate) => (Promise<DBCreate> | DBCreate),
        shapeUpdate: (userId: string, update: GraphQLUpdate) => (Promise<DBUpdate> | DBUpdate),
    } | ((userId: string, data: GraphQLCreate | GraphQLUpdate, isAdd: boolean) => (Promise<DBCreate | DBUpdate> | DBCreate | DBUpdate)) | undefined,
    /**
     * The id of the user performing the operation. Relationship building is only used when performing 
     * create, update, and delete operations, so id is always required
     */
    userId: string,
    /**
     * If relationship is a join table, data required to create the join table record
     * 
     * NOTE: Does not differentiate between a disconnect and a delete. How these are handled is 
     * determined by the database cascading.
     */
    joinData?: {
        fieldName: string, // e.g. organization.tags.tag => 'tag'
        uniqueFieldName: string, // e.g. organization.tags.tag => 'organization_tags_taggedid_tagTag_unique'
        childIdFieldName: string, // e.g. organization.tags.tag => 'tagTag'
        parentIdFieldName: string, // e.g. organization.tags.tag => 'taggedId'
        parentId: string | null, // Only needed if not a create
    }
}

type OneOrArray<T> = T | T[];

/**
 * Converts an add or update's data to proper Prisma format. 
 * 
 * NOTE1: Must authenticate before calling this function!
 * 
 * NOTE2: Only goes one layer deep. You must handle grandchildren, great-grandchildren, etc. yourself
 */
export const relationshipBuilderHelper = async<IDField extends string, GraphQLCreate extends { [x: string]: any }, GraphQLUpdate extends { [x: string]: any }, DBCreate extends { [x: string]: any }, DBUpdate extends { [x: string]: any }>({
    data,
    relationshipName,
    isAdd,
    isTransferable = true,
    isOneToOne = false,
    fieldExcludes = [],
    relExcludes = [],
    softDelete = false,
    idField = 'id' as IDField,
    shape,
    userId,
    joinData,
}: RelationshipBuilderHelperArgs<IDField, GraphQLCreate, GraphQLUpdate, DBCreate, DBUpdate>): Promise<{
    connect?: OneOrArray<{ [key in IDField]: string }>,
    disconnect?: OneOrArray<{ [key in IDField]: string }>,
    delete?: OneOrArray<{ [key in IDField]: string }>,
    create?: OneOrArray<{ [x: string]: any }>,
    update?: OneOrArray<{ where: { [key in IDField]: string }, data: { [x: string]: any } }>,
} | undefined> => {
    // Determine valid operations, and remove operations that should be excluded
    let ops = isAdd ? [RelationshipTypes.connect, RelationshipTypes.create] : Object.values(RelationshipTypes);
    if (!isTransferable) ops = difference(ops, [RelationshipTypes.connect, RelationshipTypes.disconnect]);
    ops = difference(ops, relExcludes);
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
        const shapedData = shapeRelationshipData(value, fieldExcludes, isOneToOne);
        // Should be an array if not one-to-one
        if (!isOneToOne) {
            converted[currOp] = Array.isArray(converted[currOp]) ? [...converted[currOp], ...shapedData] : shapedData;
        } else {
            converted[currOp] = shapedData;
        }
    };
    // Connects, diconnects, and deletes must be shaped in the form of { id: '123' } (i.e. no other fields)
    if (Array.isArray(converted.connect) && converted.connect.length > 0) converted.connect = converted.connect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.disconnect) && converted.disconnect.length > 0) converted.disconnect = converted.disconnect.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    if (Array.isArray(converted.delete) && converted.delete.length > 0) converted.delete = converted.delete.map((e: { [x: string]: any }) => ({ [idField]: e[idField] }));
    // Updates must be shaped in the form of { where: { id: '123' }, data: {...}}
    if (Array.isArray(converted.update) && converted.update.length > 0) {
        converted.update = converted.update.map((e: any) => ({ where: { id: e.id }, data: e }));
    }
    // Shape creates and updates
    const shapeCreate = shape !== undefined ? typeof shape === 'function' ? shape : shape.shapeCreate : undefined;
    if (shapeCreate) {
        if (Array.isArray(converted.create)) {
            const shaped: { [x: string]: any }[] = [];
            for (const create of converted.create) {
                const shapedCreate = await shapeCreate(userId, create, true);
                shaped.push(shapedCreate);
            }
            converted.create = shaped;
        }
    }
    const shapeUpdate = shape !== undefined ? typeof shape === 'function' ? shape : shape.shapeUpdate : undefined;
    if (shapeUpdate) {
        if (Array.isArray(converted.update)) {
            const shaped: { where: { [key in IDField]: string }, data: { [x: string]: any } }[] = [];
            for (const update of converted.update) {
                const shapedUpdate = await shapeUpdate(userId, update.data, false);
                shaped.push({ where: update.where, data: shapedUpdate });
            }
            converted.update = shaped;
        }
    }
    // Handle join table, if applicable
    if (joinData) {
        if (converted.connect) {
            // ex: create: [ { tag: { connect: { id: 'asdf' } } } ] <-- A join table always creates on connects
            for (const id of (converted?.connect ?? [])) {
                const curr = { [joinData.fieldName]: { connect: id } };
                converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
            }
        }
        if (converted.disconnect) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ] <-- A join table always deletes on disconnects
            for (const id of (converted?.disconnect ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[idField], [joinData.parentIdFieldName]: joinData.parentId } };
                converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
            }
        }
        if (converted.delete) {
            // delete: [ { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } } ]
            for (const id of (converted?.delete ?? [])) {
                const curr = { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: id[idField], [joinData.parentIdFieldName]: joinData.parentId } };
                converted.delete = Array.isArray(converted.delete) ? [...converted.delete, curr] : [curr];
            }
        }
        if (converted.create) {
            // ex: create: [ { tag: { create: { id: 'asdf' } } } ]
            for (const id of (converted?.create ?? [])) {
                const curr = { [joinData.fieldName]: { create: id } };
                converted.create = Array.isArray(converted.create) ? [...converted.create, curr] : [curr];
            }
        }
        if (converted.update) {
            // ex: update: [{ 
            //         where: { organization_tags_taggedid_tagTag_unique: { tagTag: 'asdf', taggedId: 'fdas' } },
            //         data: { tag: { update: { tag: 'fdas', } } }
            //     }]
            for (const data of (converted?.update ?? [])) {
                const curr = {
                    where: { [joinData.uniqueFieldName]: { [joinData.childIdFieldName]: data.where[idField], [joinData.parentIdFieldName]: joinData.parentId } },
                    data: { [joinData.fieldName]: { update: data.data } }
                };
                converted.update = Array.isArray(converted.update) ? [...converted.update, curr] : [curr];
            }
        }
    }
    return Object.keys(converted).length > 0 ? converted : undefined;
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
    const formatter: FormatConverter<any, any> | undefined = typeof query?.__typename === 'string' ? ObjectMap[query.__typename as keyof typeof ObjectMap]?.format : undefined;
    if (formatter) {
        // Remove calculated fields from query, since these will not be in obj
        formattedQuery = formatter?.removeSupplementalFields ? formatter.removeSupplementalFields(query) : query;
    }
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
 * Finds current userId in Request object
 * @param req Request object
 * @returns First userId in Session object, or null if not found/invalid
 */
export const getUserId = (req: ReqForUserAuth): string | null => {
    if (!req || !Array.isArray(req?.users) || req.users.length === 0) return null;
    const userId = req.users[0].id;
    return typeof userId === 'string' && uuidValidate(userId) ? userId : null;
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

type AddSupplementalFieldsHelper<GraphQLModel> = {
    objects: ({ id: string } & { [x: string]: any })[],
    partial: PartialGraphQLInfo,
    resolvers: [keyof GraphQLModel, (ids: string[]) => Promise<any>][],
}

/**
 * Helper function for simplifying addSupplementalFields
 */
export async function addSupplementalFieldsHelper<GraphQLModel>({
    objects,
    partial,
    resolvers,
}: AddSupplementalFieldsHelper<GraphQLModel>): Promise<RecursivePartial<GraphQLModel>[]> {
    if (!objects || objects.length === 0) return [];
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Get supplemental fields, and inject into objects
    for (const [field, resolver] of resolvers) {
        // If not in partial, skip
        if (!partial[field as string]) continue;
        const supplemental = await resolver(ids);
        objects = objects.map((x, i) => ({ ...x, [field]: supplemental[i] }));
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

type ExistsArray = {
    ids: (string | null | undefined)[],
    prismaDelegate: any,
    where: { [x: string]: any },
}

/**
 * Helper function for querying a list of objects using a specified search query
 * @returns Array in the same order as the ids, with a boolean indicating whether the object was found
 */
export async function existsArray({ ids, prismaDelegate, where }: ExistsArray): Promise<Array<boolean>> {
    if (ids.length === 0) return [];
    // Take out nulls
    const idsToQuery = onlyValidIds(ids);
    // Query
    const objects = await prismaDelegate.findMany({
        where,
        select: { id: true },
    })
    // Convert to array of booleans
    return idsToQuery.map(id => objects.some(({ id: objectId }: { id: string }) => objectId === id));
}

/**
 * Helper function for combining Prisma queries. This is basically a spread, 
 * but it also combines AND, OR, and NOT queries
 * @param queries Array of query objects to combine
 * @returns Combined query object, with all fields combined
 */
export function combineQueries(queries: ({ [x: string]: any } | null | undefined)[]): { [x: string]: any } {
    const combined: { [x: string]: any } = {};
    for (const query of queries) {
        if (!query) continue;
        for (const [key, value] of Object.entries(query)) {
            let currValue = value;
            // If key is AND, OR, or NOT, combine
            if (['AND', 'OR', 'NOT'].includes(key)) {
                // Value should be an array
                if (!Array.isArray(value)) {
                    currValue = [value];
                }
                // For AND, combine arrays
                if (key === 'AND') {
                    combined[key] = key in combined ? [...combined[key], ...currValue] : currValue;
                }
                // For OR and NOT, set as value if none exists
                else if (!(key in combined)) {
                    combined[key] = currValue;
                }
                // Otherwise, combine values using AND. This is because we can't have duplicate keys
                else {
                    // Store temp value 
                    const temp = combined[key];
                    // Delete key
                    delete combined[key];
                    // Add old and new value to AND array
                    combined.AND = [
                        ...(combined.AND || []),
                        { [key]: temp },
                        { [key]: currValue },
                    ];
                }
            }
            // Otherwise, just add it
            else combined[key] = value;
        }
    }
    return combined;
}

type ExceptionsBuilderProps = {
    /**
     * Fields that are allowed to be queried. Supports nested fields through dot notation
     */
    canQuery: string[],
    /**
     * Default for main field
     */
    defaultValue?: any,
    /**
     * Field to check for stringified exceptions
     */
    exceptionField: string,
    /**
     * Input object, with exceptions in one of the fields
     */
    input: { [x: string]: any },
    /**
     * Main field being queried
     */
    mainField: string,
}

/**
 * Assembles custom query exceptions (i.e. query has some condition OR <exceptions>). 
 * If an 'id' field is allowed (e.g. 'parent.id') and the current value is a string, then we treat as 
 * a 'connect' query (i.e. assume that the string is a primary key for the object)
 */
export function exceptionsBuilder({
    canQuery,
    defaultValue,
    exceptionField,
    input,
    mainField,
}: ExceptionsBuilderProps): { [x: string]: any } {
    // Initialize result
    const result: { [x: string]: any } = { [mainField]: input[mainField] ?? defaultValue };
    // Helper function for checking if a stringified object is a primitive or an array of primitives.
    // Returns boolean indicating whether it is a primitive, and the parsed object
    const getPrimitive = (x: string): [boolean, any] => {
        const primitiveCheck = (y: any): boolean => { return y === null || typeof y === 'string' || typeof y === 'number' || typeof y === 'boolean' };
        let value: any;
        try { value = JSON.parse(x); }
        catch (err) { return [false, undefined]; }
        if (Array.isArray(value)) {
            if (value.every(primitiveCheck)) return [true, value]
        }
        else if (primitiveCheck(value)) return [true, value];
        return [false, value];
    }
    /**
     * Helper function for converting a list of fields to a nested object
     * @param fields List of fields to convert
     * @param value Value to assign to the last field
     */
    const fieldsToObject = (fields: string[], value: any): { [x: string]: any } => {
        if (fields.length === 0) return value;
        const [field, ...rest] = fields;
        return { [field]: fieldsToObject(rest, value) };
    }
    /**
     * Helper function to add an object to the result's OR array
     * @param allowed Fields that are allowed to be queried
     * @param field Field's name
     * @param value Field's stringified value
     * @param recursedFields Nested fields in current recursion. These are used to generated nested queries
     */
    const addToOr = (allowed: string[], field: string, value: string, recursedFields: string[] = []): void => {
        const [isPrimitive, parsedValue] = getPrimitive(value);
        // Check if field is allowed
        if (isPrimitive && allowed.includes(field)) {
            // If not array, add to result
            if (!Array.isArray(parsedValue)) result.OR.push(fieldsToObject([...recursedFields, field], parsedValue));
            // Otherwise, wrap in { in: } and add to result
            else result.OR.push(fieldsToObject([...recursedFields, field], { in: parsedValue }));
        }
        // Check if field is allowed with 'id' appended
        else if (allowed.includes(`${field}.id`)) {
            // If not array, add to result
            if (!Array.isArray(parsedValue) && typeof parsedValue === 'string') result.OR.push(fieldsToObject([...recursedFields, field, 'id'], parsedValue));
            // Otherwise, wrap in { in: } and add to result
            else if (Array.isArray(parsedValue) && parsedValue.every(x => typeof x === 'string')) result.OR.push(fieldsToObject([...recursedFields, field, 'id'], { in: parsedValue }));
        }
        // Otherwise, check if we should recurse
        else if (typeof parsedValue === 'object' && field in parsedValue) {
            const matchingFields = allowed.filter(x => x.startsWith(`${field}.`));
            if (matchingFields.length > 0) {
                addToOr(
                    allowed.filter(x => x.startsWith(`${field}.`)),
                    field,
                    JSON.stringify(parsedValue[field]),
                    [...recursedFields, field],
                );
            }
        }
    }
    if (!(typeof input === 'object' && mainField in input)) return result;
    // Get mainField value
    // If exceptionField is present, wrap in OR
    if (exceptionField in input) {
        result.OR = [{ [mainField]: result[mainField] }];
        delete result[mainField];
        // If exceptionField is an array, add each item to OR
        if (Array.isArray(input[exceptionField])) {
            // Delete mainField from result, since it will be in OR
            for (const exception of input[exceptionField]) {
                addToOr(canQuery, lowercaseFirstLetter(exception.field), exception.value);
            }
        }
        // Otherwise, add exceptionField to OR
        else {
            addToOr(canQuery, lowercaseFirstLetter(input[exceptionField].field), input[exceptionField].value);
        }
    }
    return result;
}

type VisibilityBuilderProps<GraphQLModelType> = {
    model: ModelLogic<GraphQLModelType, any, any, any>,
    userId: string | null | undefined,
    visibility?: VisibilityType | null | undefined,
}

/**
 * Assembles visibility query
 */
export function visibilityBuilder<GraphQLModelType>({
    model,
    userId,
    visibility,
}: VisibilityBuilderProps<GraphQLModelType>): { [x: string]: any } {
    // If visibility is set to public or not defined, 
    // or user is not logged in, or model does not have 
    // the correct data to query for ownership
    if (!visibility || visibility === VisibilityType.Public || !userId || !model.permissions) {
        return { isPrivate: false };
    }
    // If visibility is set to private, query private objects that you own
    else if (visibility === VisibilityType.Private) {
        return combineQueries([{ isPrivate: true }, model.permissions().ownershipQuery(userId)])
    }
    // Otherwise, must be set to All
    else {
        let query: { [x: string]: any } = model.permissions().ownershipQuery(userId);
        // If query has OR field with an array value, add isPrivate: false to array
        if ('OR' in query && Array.isArray(query.OR)) {
            query.OR.push({ isPrivate: false });
        }
        // Otherwise, wrap query in OR with isPrivate: false
        else {
            query = { OR: [query, { isPrivate: false }] };
        }
        return query;
    }
}

type PermissionsBuilderProps<GraphQLModelType> = {
    model: ModelLogic<GraphQLModelType, any, any, any>,
    userId: string | null | undefined,
    visibility?: VisibilityType | null | undefined,
}

/**
 * Assembles permissions query, which is used to find all data required to determine if a user has 
 * the correct permissions to perform an action on a model (e.g. create, update, view, fork)
 */
export function permissionsBuilder<GraphQLModelType>({
    model,
    userId,
    visibility,
}: PermissionsBuilderProps<GraphQLModelType>): { [x: string]: any } {
    fdsafds
}