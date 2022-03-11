import { CODE, resourceCreate, resourceTranslationCreate, resourceTranslationsUpdate, resourceUpdate } from "@local/shared";
import { Resource, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, Count, ResourceFor } from "../schema/types";
import { PrismaType } from "types";
import { CUDInput, CUDResult, FormatConverter, GraphQLModelType, modelToGraphQL, relationshipToPrisma, RelationshipTypes, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { CustomError } from "../error";
import _ from "lodash";
import { TranslationModel } from "./translation";

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
        const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
        return { ...forQuery, ...languagesQuery };
    },
})

// Maps resource for type to correct field
const forMap = {
    [ResourceFor.Organization]: 'organizationId',
    [ResourceFor.Project]: 'projectId',
    [ResourceFor.Routine]: 'routineId',
    [ResourceFor.User]: 'userId',
}

export const resourceMutater = (prisma: PrismaType) => ({
    async toDBShape(userId: string | null, data: ResourceCreateInput | ResourceUpdateInput): Promise<any> {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...rest } = data;
        // Map forId to correct field
        if (createdFor) {
            return {
                ...rest,
                organizationId: null,
                projectId: null,
                routineId: null,
                userId: null,
                translations: TranslationModel().relationshipBuilder(userId, data, { create: resourceTranslationCreate, update: resourceTranslationsUpdate }, false),
                [forMap[createdFor]]: createdForId
            };
        }
        return rest;
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
        relationshipName: string = 'resources',
    ): Promise<{ [x: string]: any } | undefined> {
        console.log('in resource relationshipbuilder start')
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName, isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        console.log('in resource relationshipbuilder formattedInput. going to validate...', formattedInput)
        await this.validateMutations({
            userId,
            createMany: createMany as ResourceCreateInput[],
            updateMany: updateMany as ResourceUpdateInput[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
            // If title or description is not provided, try querying for the link's og tags TODO
            formattedInput.create = formattedInput.create.map(async (data) => await this.toDBShape(userId, data as any));
        }
        if (Array.isArray(formattedInput.update)) {
            formattedInput.update = formattedInput.update.map(async (data) => await this.toDBShape(userId, data as any));
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async validateMutations({
        userId, createMany, updateMany, deleteMany
    }: ValidateMutationsInput<ResourceCreateInput, ResourceUpdateInput>): Promise<void> {
        if ((createMany || updateMany || deleteMany) && !userId) throw new CustomError(CODE.Unauthorized, 'User must be logged in to perform CRUD operations');
        console.log('in resource validateMutations...')
        // TODO check that user can add resource to this forId, like in node validateMutations
        if (createMany) {
            console.log('node validate createMany', createMany);
            createMany.forEach(input => resourceCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => TranslationModel().profanityCheck(input));
            // Check for max resources on object TODO
        }
        if (updateMany) {
            console.log('node validate updateMany', updateMany);
            updateMany.forEach(input => resourceUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => TranslationModel().profanityCheck(input));
        }
        console.log('finishedd resource validateMutations :)')
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
                const data = await this.toDBShape(userId, input);
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
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                // Find in database
                let object = await prisma.report.findFirst({
                    where: {
                        id: input.id,
                        userId,
                    }
                })
                if (!object) throw new CustomError(CODE.ErrorUnknown);
                // Update object
                const currUpdated = await prisma.resource.update({
                    where: { id: object.id },
                    data,
                    ...selectHelper(partial)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, partial);
                // Add to updated array
                updated = updated ? [...updated, converted] : [converted];
            }
        }
        if (deleteMany) {
            deleted = await prisma.report.deleteMany({
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