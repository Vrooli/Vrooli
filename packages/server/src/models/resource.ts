import { resourceCreate, resourcesCreate, resourcesUpdate, resourceUpdate } from "@shared/validation";
import { CODE, ResourceSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper } from "./builder";
import { TranslationModel } from "./translation";
import { OrganizationModel, organizationQuerier } from "./organization";
import { ProjectModel } from "./project";
import { RoutineModel } from "./routine";
import { CustomError, genErrorCode } from "../events";
import { Resource, ResourceSearchInput, ResourceCreateInput, ResourceUpdateInput, Count } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

export const resourceFormatter = (): FormatConverter<Resource, any> => ({
    relationshipMap: { '__typename': 'Resource' }, // For now, resource is never queried directly. So no need to handle relationships
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
        return getSearchStringQueryHelper({
            searchString,
            resolver: ({ insensitive }) => ({
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                    { link: { ...insensitive } },
                ]
            })
        })
    },
    customQueries(input: ResourceSearchInput): { [x: string]: any } {
        // const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

// TODO create proper permissioner
export const resourcePermissioner = () => ({
    ownershipQuery: (userId: string) => ({
        OR: [
            organizationQuerier().hasRoleInOrganizationQuery(userId),
            { user: { id: userId } }
        ]
    })
})

export const resourceMutater = (prisma: PrismaType) => ({
    /**
     * Verify that the user can add these resources to the given object
     */
    async authorizedAdd(userId: string, resources: ResourceCreateInput[], prisma: PrismaType): Promise<void> {
        //TODO
    },
    /**
     * Verify that the user can update/delete these resources to the given object
     */
    async authorizedUpdateOrDelete(userId: string, resourceIds: string[], prisma: PrismaType): Promise<void> {
        // Query all resources to find their owner
        const existingResources = await prisma.resource.findMany({
            where: { id: { in: resourceIds } },
            select: {
                id: true,
                list: {
                    select: {
                        apiVersion: { select: { id: true } },
                        organization: { select: { id: true } },
                        post: { select: { id: true } },
                        project: { select: { id: true } },
                        routineVersion: { select: { id: true } },
                        smartContractVersion: { select: { id: true } },
                        standardVersion: { select: { id: true } },
                        userSchedule: { select: { userId: true } },
                    }
                }
            }
        })
        // Check if user is owner of all resources
        const isOwner = async (userId: string, data: any) => {
            if (!data.list) return false;
            // if (data.list.apiVersion) {
            //     const permissions = await ApiModel.permissions().get({ objects: [data.list.apiVersion], prisma, userId });
            //     return permissions[0].canEdit;
            // }
            if (data.list.organization) {
                const permissions = await OrganizationModel.permissions().get({ objects: [data.list.organization], prisma, userId });
                return permissions[0].canEdit;
            }
            // else if (data.list.post) {
            //     const permissions = await PostModel.permissions().get({ objects: [data.list.post], prisma, userId });
            //     return permissions[0].canEdit;
            // }
            else if (data.list.project) {
                const permissions = await ProjectModel.permissions().get({ objects: [data.list.project], prisma, userId });
                return permissions[0].canEdit
            }
            else if (data.list.routineVersion) {
                const permissions = await RoutineModel.permissions().get({ objects: [data.list.routineVersion], prisma, userId });
                return permissions[0].canEdit;
            }
            // else if (data.list.smartContractVersion) {
            //     const permissions = await SmartContractModel.permissions().get({ objects: [data.list.smartContractVersion], prisma, userId });
            //     return permissions[0].canEdit;
            // }
            // else if (data.list.standardVersion) {
            //     const permissions = await StandardModel.permissions().get({ objects: [data.list.standardVersion], prisma, userId });
            //     return permissions[0].canEdit;
            // }
            else if (data.list.userSchedule) {
                return data.list.userSchedule.userId === userId;
            }
            return false;
        }
        if (!existingResources.some(r => isOwner(userId, r)))
            throw new CustomError(CODE.Unauthorized, 'You do not own the resource, or are not an admin of its organization', { code: genErrorCode('0086') });
    },
    toDBBase(userId: string, data: ResourceCreateInput | ResourceUpdateInput, isAdd: boolean) {
        return {
            id: data.id,
            index: data.index,
            translations: TranslationModel.relationshipBuilder(userId, data, { create: resourceCreate, update: resourceUpdate }, isAdd),
        };
    },
    toDBRelationshipCreate(userId: string, data: ResourceCreateInput): Prisma.resourceCreateWithoutListInput {
        return {
            ...this.toDBBase(userId, data, true),
            link: data.link,
            usedFor: data.usedFor,
        };
    },
    toDBRelationshipUpdate(userId: string, data: ResourceUpdateInput): Prisma.resourceUpdateWithoutListInput {
        return {
            ...this.toDBBase(userId, data, false),
            link: data.link ?? undefined,
            usedFor: data.usedFor ?? undefined,
        };
    },
    toDBCreate(userId: string, data: ResourceCreateInput & { listId: string }): Prisma.resourceUpsertArgs['create'] {
        return {
            ...this.toDBRelationshipCreate(userId, data),
            listId: data.listId,
        };
    },
    toDBUpdate(userId: string, data: ResourceUpdateInput): Prisma.resourceUpsertArgs['update'] {
        return {
            ...this.toDBRelationshipUpdate(userId, data),
        };
    },
    async relationshipBuilder(
        userId: string,
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
            const creates: { [x: string]: any }[] = [];
            for (const create of formattedInput.create) {
                creates.push(this.toDBRelationshipCreate(userId, create as any));
            }
            formattedInput.create = creates;
        }
        if (Array.isArray(formattedInput.update)) {
            const updates: { where: { [x: string]: any }, data: { [x: string]: any } }[] = [];
            for (const update of formattedInput.update) {
                updates.push({
                    where: update.where,
                    data: this.toDBRelationshipUpdate(userId, update.data as any),
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
        // Check for max resources on object TODO
        if (createMany) {
            resourcesCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel.profanityCheck(createMany);
            await this.authorizedAdd(userId as string, createMany, prisma);
        }
        if (updateMany) {
            resourcesUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(updateMany.map(u => u.data));
            await this.authorizedUpdateOrDelete(userId as string, updateMany.map(u => u.where.id), prisma);
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<ResourceCreateInput & { listId: string }, ResourceUpdateInput>): Promise<CUDResult<Resource>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // If title or description is not provided, try querying for the link's og tags TODO
                // Call createData helper function
                const data = this.toDBCreate(userId, input);
                // Create object
                const currCreated = await prisma.resource.create({ data, ...selectHelper(partialInfo) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, partialInfo);
                // Add to created array
                created = created ? [...created, converted] : [converted];
            }
        }
        if (updateMany) {
            // Loop through each update input
            for (const input of updateMany) {
                // Find in database
                let object = await prisma.resource.findFirst({
                    where: {
                        AND: [
                            input.where,
                            { list: { userId } },
                        ]
                    }
                })
                if (!object)
                    throw new CustomError(CODE.ErrorUnknown, 'Resource not found', { code: genErrorCode('0088') });
                // Update object
                const currUpdated = await prisma.resource.update({
                    where: input.where,
                    data: this.toDBUpdate(userId, input.data),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.resource.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { list: resourcePermissioner().ownershipQuery(userId as string) },
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

export const ResourceModel = ({
    prismaObject: (prisma: PrismaType) => prisma.resource,
    format: resourceFormatter(),
    mutate: resourceMutater,
    search: resourceSearcher(),
    type: 'Resource' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================