// Components for providing basic functionality to model objects
import { CODE } from '@local/shared';
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, PageInfo, InputMaybe, ReportInput, Scalars, TimeFrame } from '../schema/types';
import { CustomError } from '../error';
import { PrismaType, RecursivePartial } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';
import graphqlFields from 'graphql-fields';
import pkg from 'lodash';
const { isObject } = pkg;


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

export type BaseType = PrismaModels['comment']; // It doesn't matter what PrismaType is used here, it's just to help TypeScript handle Prisma operations

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
 * Adds "select" to the correct of an object to make it a Prisma select
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

// /**
//  * Creates or updates a one-to-many relationship between two objects 
//  * (where the child has a field that references the parent's id). 
//  * @param modelName The name of the child model in Prisma
//  * @param parentIdFieldName The name of the column in the child model that references the parent's id
//  * @param prisma The Prisma client
//  * @param childData The child data
//  * @param parentId The id of the parent object, if updating an existing relationship
//  */
// export async function upsertOneToManyRelationship<Child extends { [x: string]: any }>(
//     childName: keyof PrismaType,
//     parentIdFieldName: string,
//     prisma: PrismaType,
//     childData: Child,
//     parentId?: string | null): Promise<Child> {
//     // Check arguments
//     if (!prisma) throw new CustomError(CODE.InvalidArgs);
//     // Remove the relationship's relationship data, as it is handled on a case-by-case basis for security reasons
//     const childPrimitives = onlyPrimitives(childData);
//     let result;
//     // Check for id in modelData to determine if this is an update or insert
//     if (!childPrimitives.id) {
//         // Insert relationship, with reference to model
//         result = await (prisma[childName] as BaseType).create({
//             data: {
//                 ...childPrimitives,
//                 id: undefined,
//                 [parentIdFieldName]: parentId
//             } as any
//         }) as unknown as Partial<Child> | null;
//     } else {
//         // Update relationship
//         result = await (prisma[childName] as BaseType).update({
//             where: { id: childPrimitives.id as string },
//             data: childPrimitives
//         })
//     }
//     return result as Child;
// }

// /**
//  * Creates or updates a many-to-many relationship between two objects 
//  * (where there is a joining table to link the parent and child). 
//  * @param joinTableName The name of the join table in Prisma
//  * @param joinTableUniqueName The name of the unique constraint in the join table
//  * @param parentIdFieldName The name of the column in the join table that references the parent's id
//  * @param childIdFieldName The name of the column in the join table that references the child's id
//  * @param prisma The Prisma client
//  * @param childData The child data, including the parent and child ids in the same keys as specified by parentIdFieldName and childIdFieldName
//  * @param parentId The id of the parent object
//  */
// export async function upsertManyToManyRelationship<Child extends { [x: string]: any }>(
//     joinTableName: keyof PrismaType,
//     joinTableUniqueName: string,
//     childIdFieldName: string,
//     parentIdFieldName: string,
//     prisma: PrismaType,
//     joinData: { [x: string]: any },
// ): Promise<Child> {
//     // Check arguments
//     if (!prisma) throw new CustomError(CODE.InvalidArgs);
//     // Remove the relationship's relationship data, as it is handled on a case-by-case basis for security reasons
//     const joinPrimitives = onlyPrimitives(joinData);
//     if (!joinPrimitives[parentIdFieldName] || !joinPrimitives[childIdFieldName]) throw new CustomError(CODE.InvalidArgs);
//     return await (prisma[joinTableName] as BaseType).upsert({
//         where: { [joinTableUniqueName]: { [childIdFieldName]: joinPrimitives[childIdFieldName], [parentIdFieldName]: joinPrimitives[parentIdFieldName] } },
//         create: joinPrimitives as any,
//         update: joinPrimitives as any
//     }) as unknown as Child;
// }

// /**
//  * Adds "create" field to the correct parts of an object to make it a correct Prisma data object. 
//  * Also keeps or removes specified fields, with support for nested fields (e.g. "parent.child.field").
//  * @param data The object being shaped. Should already be shaped for Prisma (i.e. not GraphQL)
//  * @param removeFields The fields to remove. If not specified, no fields are removed except if "keepFields" is specified
//  * @param keepFields The fields to keep. If not specified, all fields are kept except for ones in "removeFields"
//  */
// export const shapeCreateData = (
//     data: { [x: string]: any },
//     removeFields?: string[],
//     keepFields?: string[],
// ): { [x: string]: any } => {
//     // Create result object
//     let converted: { [x: string]: any } = {};
//     // Loop through object's keys
//     Object.keys(data).forEach((key) => {
//         console.log('hereeeeee', key, data[key]);
//         // If "__typename", skip
//         if (key === '__typename') return;
//         // Determine if this key should be kept
//         let skip = false;
//         if (Array.isArray(keepFields) && keepFields.length > 0) {
//             skip = !keepFields.some(f => f === key || f.startsWith(key + '.'));
//         }
//         if (Array.isArray(removeFields) && removeFields.length > 0) {
//             skip = skip || removeFields.some(f => f === key || f.startsWith(key + '.'));
//         }
//         if (skip) return;
//         // If value is a primitive, add to result without modification
//         if (!Array.isArray(data[key]) && !isObject(data[key])) {
//             console.log('is primitive', data[key]);
//             converted[key] = data[key];
//             return;
//         }
//         // Determine keep fields for nested object
//         let nestedKeepFields: string[] | undefined;
//         if (keepFields) {
//             nestedKeepFields = keepFields.filter(field => field.startsWith(key + "."));
//         }
//         // Determine remove fields for nested object
//         let nestedRemoveFields: string[] | undefined;
//         if (removeFields) {
//             nestedRemoveFields = removeFields.filter(field => field.startsWith(key + "."));
//         }
//         console.log('nestedRemoveFields', nestedRemoveFields);
//         console.log('nestedKeepFields', nestedKeepFields);
//         // If value is an array (i.e. many-to-many relationship), recursively shape each element in value
//         if (Array.isArray(data[key])) {
//             console.log('isArray', data[key]);
//             // Determine which elements will be created (i.e. no id)
//             const willCreate: boolean[] = data[key].map((e: any) => {
//                 // If "id" is present, but wrapped by a join table (e.g. { role: { id: '123' } })
//                 if (isObject(e) && Object.keys(e).length === 1) {
//                     const joinValue = e[Object.keys(e)[0]];
//                     if (isObject(joinValue)) return !joinValue.id;
//                 }
//                 // If "id" is present directly (e.g. { id: '123' })
//                 return !e.id
//             });
//             // Shape elements that will be created and connected
//             let create = [];
//             let connect = [];
//             for (let i = 0; i < data[key].length; i++) {
//                 if (willCreate[i]) {
//                     create.push(shapeCreateData(data[key][i], nestedRemoveFields, nestedKeepFields));
//                 } else {
//                     connect.push({ id: data[key][i].id });
//                 }
//             }
//             console.log('create', create);
//             console.log('connect', connect);
//             converted[key] = {};
//             if (create.length > 0) converted[key].create = create;
//             if (connect.length > 0) converted[key].connect = connect;
//         }
//         // If value is an object (i.e. one-to-one relationship), recursively shape value
//         else if (isObject(data[key])) {
//             console.log('isObject', data[key]);
//             const curr = data[key];
//             // Determine if object will be created or connected. If connected, store id
//             let connectId;
//             // Connect if "id" is present, but wrapped by a join table (e.g. { role: { id: '123' } })
//             if (isObject(curr) && Object.keys(curr).length === 1) {
//                 const joinValue = curr[Object.keys(curr)[0]];
//                 if (isObject(joinValue)) {
//                     connectId = joinValue.id;
//                 }
//             }
//             // Connect if "id" is present directly (e.g. { id: '123' })
//             else {
//                 connectId = curr.id;
//             }
//             if (connectId) {
//                 converted[key] = { connect: { id: connectId } };
//             } else {
//                 converted[key] = { create: shapeCreateData(data[key], nestedRemoveFields, nestedKeepFields) };
//             }
//         }
//     });
//     // The root object doesn't get wrapped in a "create" field
//     return converted;
// }

/**
 * Removes any non-primitive fields (i.e. relationships) that are not specified in the "keepFields" array.
 * @param data The object being filtered
 * @param keepFields The relationships to keep. If not specified, all fields are kept
 * @returns data without non-specified relationships
 */
export const keepOnly = (data: { [x: string]: any }, keepFields: string[]): { [x: string]: any } => {
    // Create result object
    let converted: { [x: string]: any } = {};
    // Loop through object's keys
    Object.keys(data).forEach((key) => {
        // If relationship
        if (isObject(data[key])) {
            // Only keep if key in keepFields
            if (keepFields.some(f => f === key)) {
                converted[key] = data[key];
            }
        }
        else converted[key] = data[key];
    });
    return converted;
}

/**
 * Recursively removes specified fields from an object, 
 * WHILE keeping their values
 */
export const removeFields = (
    data: { [x: string]: any },
    fields: string[],
): any => {
    if (Array.isArray(data)) {
        return data.map(e => removeFields(e, fields));
    }
    if (!isObject(data)) {
        return data;
    }
    let result: {[x: string]: any} = {}
    let found = false;
    Object.keys(data).forEach(key => {
        const curr = (data as any)[key];
        fields.forEach(field => {
            if (key === field && isObject(curr)) {
                found = true;
                result = removeFields(curr, fields);
            }
        })
        if (!found)
            result[key] = removeFields(curr, fields);
    })
    return result;
}

/**
 * Grabs relationship data from a Prisma add/update data object
 */
export const getRelationshipData = (
    data: { [x: string]: any },
    relationship: string,
): any[] => {
    // Get relationship field value
    const value = data[relationship];
    // Remove prisma operations. We want only the data!
    const shapedValue = removeFields(value, ['connect', 'create'])
    return Array.isArray(shapedValue) ? shapedValue : [];
}