import { CODE, resourceListsCreate, resourceListsUpdate, resourceListTranslationsCreate, resourceListTranslationsUpdate } from "@local/shared";
import { ResourceList, ResourceListCreateInput, ResourceListUpdateInput, Count, ResourceListSortBy, ResourceListSearchInput } from "../../schema/types";
import { PrismaType } from "types";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, relationshipToPrisma, RelationshipTypes, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../../error";
import { TranslationModel } from "./translation";
import { ResourceModel } from "./resource";
import { genErrorCode } from "../../logger";

//==============================================================
/* #region Custom Components */
//==============================================================

export const resourceListFormatter = (): FormatConverter<ResourceList> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.ResourceList,
        'resources': GraphQLModelType.Resource,
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
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { translations: { some: { language: languages ? { in: languages } : undefined, description: { ...insensitive } } } },
                { translations: { some: { language: languages ? { in: languages } : undefined, title: { ...insensitive } } } },
            ]
        })
    },
    customQueries(input: ResourceListSearchInput): { [x: string]: any } {
        return {
            ...(input.languages !== undefined ? { translations: { some: { language: { in: input.languages } } } } : {}),
        }
    },
})

// /**
//  * Maps object type to Id field
//  */
//  const resourceListMapper = {
//     [GraphQLModelType.Organization]: 'organizationId',
//     [GraphQLModelType.Project]: 'projectId',
//     [GraphQLModelType.Routine]: 'routineId',
//     [GraphQLModelType.User]: 'userId',
// }

export const resourceListMutater = (prisma: PrismaType) => ({
    async toDBShape(userId: string | null, data: ResourceListCreateInput | ResourceListUpdateInput, isAdd: boolean): Promise<any> {
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
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resourceLists',
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape. Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resource lists (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as ResourceListCreateInput[],
            updateMany: updateMany as { where: { id: string }, data: ResourceListUpdateInput }[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
            // If title or description is not provided, try querying for the link's og tags TODO
            const creates = [];
            for (const create of formattedInput.create) {
                creates.push(await this.toDBShape(userId, create as any, true));
            }
            formattedInput.create = creates;
        }
        if (Array.isArray(formattedInput.update)) {
            const updates = [];
            for (const update of formattedInput.update) {
                updates.push({
                    where: update.where,
                    data: await this.toDBShape(userId, update.data as any, false),
                })
            }
            formattedInput.update = updates;
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ResourceListCreateInput, ResourceListUpdateInput>): Promise<void> {
        if (!createMany && !updateMany && !deleteMany) return;
        if (!userId)
            throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations', { code: genErrorCode('0089') });
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
                const data = await this.toDBShape(userId, input, true);
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
                    data: await this.toDBShape(userId, input.data, false),
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
})

//==============================================================
/* #endregion Model */
//==============================================================