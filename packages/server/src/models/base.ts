// Components for providing basic functionality to model objects
import { CODE } from '@local/shared';
import { PrismaSelect } from '@paljs/plugins';
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, InputMaybe, ReportInput, Scalars } from '../schema/types';
import { CustomError } from '../error';
import { PrismaType, RecursivePartial } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';


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
export type FormatConverter<GraphQLModel, DBModel> = {
    joinMapper?: JoinMap;
    toDB: (obj: RecursivePartial<GraphQLModel>) => RecursivePartial<DBModel>;
    toGraphQL: (obj: RecursivePartial<DBModel>) => RecursivePartial<GraphQLModel>;
}

/**
 * Mapper for associating a model's many-to-many relationship names with
 * their corresponding join table names.
 */
export type JoinMap = { [key: string]: string };

type BaseType = PrismaModels['comment']; // It doesn't matter what PrismaType is used here, it's just to help TypeScript handle Prisma operations
export interface BaseState<GraphQLModel> {
    prisma?: PrismaType;
    model: keyof PrismaModels;
    format: FormatConverter<GraphQLModel, any>;
}

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

type InfoType = GraphQLResolveInfo | { select: any } | null;

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
 * Helper function for creating a Prisma select object
 * @returns select object for Prisma operations
 */
export const selectHelper = (info: InfoType): any => {
    // Return undefined if info not set
    if (!info) return undefined;
    // If info contains the "select" field, then it is already formatted correctly
    if (Object.hasOwnProperty.call(info, 'select')) return info;
    // Otherwise, use PrismaSelect to format the select object
    return new PrismaSelect(info as GraphQLResolveInfo).value;
}

/**
 * Compositional component for models which can be queried by ID
 * @param state 
 * @returns 
 */
export const findByIder = <Model>({ prisma, model, format }: BaseState<Model>) => ({
    async findById(input: FindByIdInput, info: InfoType): Promise<RecursivePartial<Model> | null> {
        // Check for valid arguments
        if (!input.id || !prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper(info);
        // Access database
        const object = await (prisma[model] as BaseType).findUnique({ where: { id: input.id }, ...select }) as unknown as Partial<Model>;
        // Format object to GraphQL type
        return object ? format.toGraphQL(object) as RecursivePartial<Model> : null;
    }
})

/**
 * Compositional component for models which can be created directly from an input
 * NOTE: This is only a basic implementation, and therefore does not handle relationships
 * @param state 
 * @returns 
 */
export const creater = <ModelInput, Model>({ prisma, model, format }: BaseState<Model>) => ({
    async create(input: ModelInput, info: InfoType): Promise<RecursivePartial<Model>> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper(info);
        // Access database
        const object = await (prisma[model] as BaseType).create({ data: { ...input }, ...select }) as unknown as RecursivePartial<Model>;
        // Format object to GraphQL type
        return format.toGraphQL(object) as RecursivePartial<Model>;
    }
})

/**
 * Compositional component for models which can be updated directly from an input
 * @param state 
 * @returns 
 */
export const updater = <ModelInput extends UpdateInterface, Model>({ prisma, model, format }: BaseState<Model>) => ({
    async update(input: ModelInput, info: InfoType): Promise<RecursivePartial<Model>> {
        // Check for valid arguments
        if (!input.id || !prisma) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper(info);
        // Access database
        const object = await (prisma[model] as BaseType).update({ where: { id: input.id }, data: { ...input }, ...select }) as unknown as RecursivePartial<Model>;
        // Format object to GraphQL type
        return format.toGraphQL(object) as RecursivePartial<Model>;
    }
})

/**
 * Compositional component for models which can be deleted directly.
 * NOTE: In most situations, deletes should be wrapped in another function for checking
 * if the delete is allowed.
 * @param state 
 * @returns 
 */
export const deleter = ({ prisma, model }: BaseState<any>) => ({
    // Delete a single object
    async delete(input: DeleteOneInput): Promise<boolean> {
        // Check for valid arguments
        if (!input.id || !prisma) throw new CustomError(CODE.InvalidArgs);
        // Access database
        return await (prisma[model] as BaseType).delete({ where: { id: input.id } }) as unknown as boolean;
    },
    // Delete many objects
    async deleteMany(input: DeleteManyInput): Promise<Count> {
        // Check for valid arguments
        if (!input.ids || !prisma) throw new CustomError(CODE.InvalidArgs);
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