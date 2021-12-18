// Components for providing basic functionality to model objects
import { CODE } from '@local/shared';
import { PrismaSelect } from '@paljs/plugins';
import { Count, DeleteManyInput, DeleteOneInput, FindByIdInput, InputMaybe, ReportInput, Scalars } from '../schema/types';
import { CustomError } from '../error';
import { PrismaType } from '../types';
import { Prisma } from '@prisma/client';
import { GraphQLResolveInfo } from 'graphql';

interface InputInterface {
    id?: InputMaybe<Scalars['ID']>;
}

type BaseType = PrismaModels['comment']; // It doesn't matter what PrismaType is used here, it's just to help TypeScript handle Prisma operations
interface State {
    prisma: PrismaType;
    model: keyof PrismaModels;
}

// Strings for accessing model functions from Prisma
export const MODEL_TYPES = {
    Comment: 'comment',
    Email: 'email',
    Node: 'node',
    Organization: 'organization',
    Project: 'project',
    Resource: 'resource',
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
    [MODEL_TYPES.Routine]: Prisma.routineDelegate<T>;
    [MODEL_TYPES.Standard]: Prisma.standardDelegate<T>;
    [MODEL_TYPES.Tag]: Prisma.tagDelegate<T>;
    [MODEL_TYPES.User]: Prisma.userDelegate<T>;
}
export type PrismaModels = Models<Prisma.RejectOnNotFound | Prisma.RejectPerOperation | undefined>

type InfoType = GraphQLResolveInfo | { select: any };

/**
 * Helper function for creating a Prisma select object
 * @returns select object for Prisma operations
 */
export const selectHelper = (info: InfoType): any => {
    // Return undefined if info not set
    if (!Boolean(info)) return undefined;
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
export const findByIder = <Model>({ prisma, model }: State) => ({
    async findById(input: FindByIdInput, info: InfoType): Promise<Model | null> {
        // Check for valid arguments
        if (!Boolean(input.id)) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper(info);
        // Access database
        return await (prisma[model] as BaseType).findUnique({ where: { id: input.id }, ...select }) as Model | null;
    }
})

/**
 * Compositional component for models which can be created directly from an input
 * NOTE: This is only a basic implementation, and therefore does not handle relationships
 * @param state 
 * @returns 
 */
export const creater = <ModelInput, Model>({ prisma, model }: State) => ({
    async create(input: ModelInput, info: InfoType): Promise<Model> {
        // Create selector
        const select = selectHelper(info);
        // Access database
        return await (prisma[model] as BaseType).create({ data: { ...input }, ...select }) as unknown as Model;
    }
})

/**
 * Compositional component for models which can be updated directly from an input
 * @param state 
 * @returns 
 */
export const updater = <ModelInput extends InputInterface, Model>({ prisma, model }: State) => ({
    async update(input: ModelInput, info: InfoType): Promise<Model> {
        // Check for valid arguments
        if (!Boolean(input.id)) throw new CustomError(CODE.InvalidArgs);
        // Create selector
        const select = selectHelper(info);
        // Access database
        return await (prisma[model] as BaseType).update({ where: { id: input.id }, data: { ...input }, ...select }) as unknown as Model;
    }
})

/**
 * Compositional component for models which can be deleted directly.
 * NOTE: In most situations, deletes should be wrapped in another function for checking
 * if the delete is allowed.
 * @param state 
 * @returns 
 */
export const deleter = ({ prisma, model }: State) => ({
    // Delete a single object
    async delete(input: DeleteOneInput): Promise<boolean> {
        // Check for valid arguments
        if (!Boolean(input.id)) throw new CustomError(CODE.InvalidArgs);
        // Access database
        return await (prisma[model] as BaseType).delete({ where: { id: input.id } }) as unknown as boolean;
    },
    // Delete many objects
    async deleteMany(input: DeleteManyInput): Promise<Count> {
        // Check for valid arguments
        if (!Boolean(input.ids)) throw new CustomError(CODE.InvalidArgs);
        // Access database
        return await (prisma[model] as BaseType).deleteMany({ where: { id: { in: input.ids } } }) as unknown as Count;
    }
})

/**
 * Compositional component for models which can be reported
 * @param state 
 * @returns 
 */
export const reporter = (state: State) => ({
    async report(input: ReportInput): Promise<boolean> {
        if (!Boolean(input.id)) throw new CustomError(CODE.InvalidArgs);
        throw new CustomError(CODE.NotImplemented);
    }
})