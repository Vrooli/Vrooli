import { CODE, resourceCreate, resourceUpdate } from "@local/shared";
import { Resource, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, Count, MemberRole } from "../../schema/types";
import { PrismaType } from "types";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, relationshipToPrisma, RelationshipTypes, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../../error";
import _ from "lodash";
import { TranslationModel } from "./translation";
import { genErrorCode } from "../../logger";

//==============================================================
/* #region Custom Components */
//==============================================================

export const resourceFormatter = (): FormatConverter<Resource> => ({
    relationshipMap: { '__typename': GraphQLModelType.Resource }, // For now, resource is never queried directly. So no need to handle relationships
})

export const resourceSearcher = (): Searcher<ResourceSearchInput> => ({
    defaultSort: ResourceSortBy.IndexAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ResourceSortBy.IndexAsc]: { index: 'asc' },
            [ResourceSortBy.IndexDesc]: { index: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                { link: { ...insensitive } },
            ]
        })
    },
    customQueries(input: ResourceSearchInput): { [x: string]: any } {
        // const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        return {
            ...(input.languages ? { translations: { some: { language: { in: input.languages } } } } : {}),
        }
    },
})

// // Maps resource for type to correct field
// const forMap = {
//     [ResourceFor.Organization]: 'organizationId',
//     [ResourceFor.Project]: 'projectId',
//     [ResourceFor.Routine]: 'routineId',
//     [ResourceFor.User]: 'userId',
// }

// Queries for finding resource ownership
const userOwnerQuery = { user: { select: { id: true } } }
const organizationOwnerQuery = { organization: { select: { members: { select: { userId: true, role: true } } } } }
const projectOwnerQuery = { project: { select: { ...userOwnerQuery, ...organizationOwnerQuery } } };
const routineOwnerQuery = { routine: { select: { ...userOwnerQuery, ...organizationOwnerQuery } } };

/**
 * Uses user ownership query to check if user is owner of resource
 * @param userId ID of user performing request
 * @param data Data shaped in the same way as userOwnerQuery
 * @returns True if user is owner of resource
 */
const isUserOwner = (userId: string, data: any) => {
    if (!data.user) return false;
    return data.user.id === userId;
}

/**
 * Uses organization ownership query to check if user is owner of resource
 * @param userId ID of user performing request
 * @param data Data shaped in the same way as organizationOwnerQuery
 * @returns True if user is owner of resource
 */
const isOrganizationOwner = (userId: string, data: any) => {
    if (!data.organization) return false;
    const member = data.organization.members.find((m: any) => m.userId === userId && [MemberRole.Admin, MemberRole.Owner].includes(m.role));
    return Boolean(member);
}

/**
 * Uses project ownership query to check if user is owner of resource
 * @param userId ID of user performing request
 * @param data Data shaped in the same way as projectOwnerQuery
 * @returns True if user is owner of resource
 */
const isProjectOwner = (userId: string, data: any) => {
    if (!data.project) return false;
    return isUserOwner(userId, data) || isOrganizationOwner(userId, data);
}

/**
 * Uses routine ownership query to check if user is owner of resource
 * @param userId ID of user performing request
 * @param data Data shaped in the same way as routineOwnerQuery
 * @returns True if user is owner of resource
 */
const isRoutineOwner = (userId: string, data: any) => {
    if (!data.routine) return false;
    return isUserOwner(userId, data) || isOrganizationOwner(userId, data);
}

export const resourceMutater = (prisma: PrismaType) => ({
    /**
     * Verify that the user can add these resources to the given project
     */
    async authorizedAdd(userId: string, resources: ResourceCreateInput[], prisma: PrismaType): Promise<void> {
        //TODO
    },
    /**
     * Verify that the user can update/delete these resources to the given project
     */
    async authorizedUpdateOrDelete(userId: string, resourceIds: string[], prisma: PrismaType): Promise<void> {
        // Query all resources to find their owner
        const existingResources = await prisma.resource.findMany({
            where: { id: { in: resourceIds } },
            select: {
                id: true,
                list: {
                    select: {
                        ...userOwnerQuery,
                        ...organizationOwnerQuery,
                        ...projectOwnerQuery,
                        ...routineOwnerQuery,
                    }
                }
            }
        })
        // Check if user is owner of all resources
        const isOwner = (userId: string, data: any) => {
            if (!data.list) return false;
            return isUserOwner(userId, data.list) || isOrganizationOwner(userId, data.list) || isProjectOwner(userId, data.list) || isRoutineOwner(userId, data.list);
        }
        if (!existingResources.some(r => isOwner(userId, r))) 
            throw new CustomError(CODE.Unauthorized, 'You do not own the resource, or are not an admin of its organization', { code: genErrorCode('0086') });
    },
    toDBShape(userId: string | null, data: ResourceCreateInput | ResourceUpdateInput, isAdd: boolean, isRelationship: boolean): any {
        return {
            listId: !isRelationship ? data.listId : undefined,
            index: data.index,
            link: data.link,
            usedFor: data.usedFor,
            translations: TranslationModel().relationshipBuilder(userId, data, { create: resourceCreate, update: resourceUpdate }, isAdd),
        };
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resources',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as ResourceCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: ResourceUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
            // If title or description is not provided, try querying for the link's og tags TODO
            const creates = [];
            for (const create of formattedInput.create) {
                creates.push(this.toDBShape(userId, create as any, isAdd, true));
            }
            formattedInput.create = creates;
        }
        if (Array.isArray(formattedInput.update)) {
            const updates = [];
            for (const update of formattedInput.update) {
                updates.push({
                    where: update.where,
                    data: this.toDBShape(userId, update.data as any, false, true),
                })
            }
            formattedInput.update = updates;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ResourceCreateInput, ResourceUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId) 
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0087') });
        if (createMany) {
            createMany.forEach(input => resourceCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            await this.authorizedAdd(userId as string, createMany, prisma);
            // Check for max resources on object TODO
        }
        if (updateMany) {
            updateMany.forEach(input => resourceUpdate.validateSync(input.data, { abortEarly: false }));
            updateMany.forEach(input => TranslationModel().profanityCheck(input.data));
            await this.authorizedUpdateOrDelete(userId as string, updateMany.map(u => u.where.id), prisma);
        }
    },
    async cud({ partial, userId, createMany, updateMany, deleteMany }: CUDInput<ResourceCreateInput, ResourceUpdateInput>): Promise<CUDResult<Resource>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // If title or description is not provided, try querying for the link's og tags TODO
                // Call createData helper function
                const data = await this.toDBShape(userId, input, true, false);
                // Create object
                const currCreated = await prisma.resource.create({ data, ...selectHelper(partial) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partial);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                console.log('in resource update', JSON.stringify(input))
                // Find in database
                let object = await prisma.resource.findFirst({
                    where: {
                        AND: [
                            input.where,
                            { list: { userId }},
                        ]
                    }
                })
                console.log('got object, ', JSON.stringify(object))
                if (!object) 
                    throw new CustomError(CODE.ErrorUnknown, 'Resource not found', { code: genErrorCode('0088') });
                // Update object
                const currUpdated = await prisma.resource.update({
                    where: input.where,
                    data: await this.toDBShape(userId, input.data, false, false),
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.resource.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { list: { userId }},
                    ]
                }
            })
        }
        return {
            created: createMany ? created : undefined,
            updated: updateMany ? updated : undefined,
            deleted: deleteMany ? deleted : undefined,
        };
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ResourceModel(prisma: PrismaType) {
    const prismaObject = prisma.resource;
    const format = resourceFormatter();
    const search = resourceSearcher();
    const mutate = resourceMutater(prisma);

    return {
        prisma,
        prismaObject,
        ...format,
        ...search,
        ...mutate,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================