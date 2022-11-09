import { resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@shared/validation";
import { CODE, ResourceListSortBy } from "@shared/consts";
import { combineQueries, getSearchStringQueryHelper, modelToGraphQL, relationshipToPrisma, RelationshipTypes, selectHelper } from "./builder";
import { TranslationModel } from "./translation";
import { ResourceModel } from "./resource";
import { CustomError, genErrorCode } from "../events";
import { ResourceList, ResourceListSearchInput, ResourceListCreateInput, ResourceListUpdateInput, Count } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, Searcher, ValidateMutationsInput, CUDInput, CUDResult, GraphQLModelType } from "./types";
import { Prisma } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

export const resourceListFormatter = (): FormatConverter<ResourceList, any> => ({
    relationshipMap: {
        '__typename': 'ResourceList',
        'resources': 'Resource',
    },
})

export const resourceListSearcher = (): Searcher<ResourceListSearchInput> => ({
    defaultSort: ResourceListSortBy.IndexAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceListSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceListSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceListSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceListSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ResourceListSortBy.IndexAsc]: { index: 'asc' },
            [ResourceListSortBy.IndexDesc]: { index: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string, languages?: string[]): any => {
        return getSearchStringQueryHelper({ searchString,
            resolver: ({ insensitive }) => ({ 
                OR: [
                    { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                    { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
                ]
            })
        })
    },
    customQueries(input: ResourceListSearchInput): { [x: string]: any } {
        return combineQueries([
            (input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        ])
    },
})

export const resourceListMutater = (prisma: PrismaType) => ({
    async toDBShapeBase(userId: string , data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean) {
        return {
            id: data.id,
            organizationId: data.organizationId ?? undefined,
            projectId: data.projectId ?? undefined,
            routineId: data.routineId ?? undefined,
            userId: data.userId ?? undefined,
            resources: await ResourceModel.mutate(prisma).relationshipBuilder(userId, data, isAdd),
            translations: TranslationModel.relationshipBuilder(userId, data, { create: resourceListTranslationsCreate, update: resourceListTranslationsUpdate }, isAdd),
        };
    },
    async toDBShapeCreate(userId: string , data: ResourceListCreateInput, isAdd: boolean): Promise<Prisma.resource_listUpsertArgs['create']> {
        return {
            ...(await this.toDBShapeBase(userId, data, isAdd)),
        };
    },
    async toDBShapeUpdate(userId: string , data: ResourceListUpdateInput, isAdd: boolean): Promise<Prisma.resource_listUpsertArgs['update']> {
        return {
            ...(await this.toDBShapeBase(userId, data, isAdd)),
        };
    },
    async relationshipBuilder(
        userId: string ,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resourceList',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape. Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resource lists (since they can only be applied to one object)
        let formattedInput: any = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as ResourceListCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: ResourceListUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create) && formattedInput.create.length > 0) {
            const create = formattedInput.create[0];
            // If title or description is not provided, try querying for the link's og tags TODO
            formattedInput.create = await this.toDBShapeCreate(userId, create, true);
        }
        if (Array.isArray(formattedInput.update) && formattedInput.update.length > 0) {
            const update = formattedInput.update[0].data;
            formattedInput.update = {
                where: update.where,
                data: await this.toDBShapeUpdate(userId, update.data, false),
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ResourceListCreateInput, ResourceListUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        // TODO check that user can add resource to this forId, like in node validateMutations
        if (createMany) {
            resourceListsCreate.validateSync(createMany, { abortEarly: false });
            TranslationModel.profanityCheck(createMany);
            // Check for max resources on object TODO
        }
        if (updateMany) {
            resourceListsUpdate.validateSync(updateMany.map(u => u.data), { abortEarly: false });
            TranslationModel.profanityCheck(updateMany.map(u => u.data));
        }
    },
    async cud({ partialInfo, userId, createMany, updateMany, deleteMany }: CUDInput<ResourceListCreateInput, ResourceListUpdateInput>): Promise<CUDResult<ResourceList>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // If title or description is not provided, try querying for the link's og tags TODO
                // Call createData helper function
                const data = await this.toDBShapeCreate(userId, input, true);
                // Create object
                const currCreated = await prisma.resource_list.create({ data, ...selectHelper(partialInfo) });
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
                let object = await prisma.report.findFirst({
                    where: { ...input.where, userId }
                })
                if (!object)
                    throw new CustomError(CODE.ErrorUnknown, 'Object not found', { code: genErrorCode('0090') });
                // Update object
                const currUpdated = await prisma.resource_list.update({
                    where: input.where,
                    data: await this.toDBShapeUpdate(userId, input.data, false),
                    ...selectHelper(partialInfo)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partialInfo);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.resource_list.deleteMany({
                where: {
                    AND: [
                        { id: { in: deleteMany } },
                        { userId },
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

export const ResourceListModel = ({
    prismaObject: (prisma: PrismaType) => prisma.resource_list,
    format: resourceListFormatter(),
    mutate: resourceListMutater,
    search: resourceListSearcher(),
    type: 'ResourceList' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================