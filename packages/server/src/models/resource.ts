import { CODE, resourceCreate, ResourceFor, resourceUpdate } from "@local/shared";
import { Resource, ResourceCountInput, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, Count } from "../schema/types";
import { PrismaType } from "types";
import { addSupplementalFields, counter, CUDInput, CUDResult, FormatConverter, modelToGraphQL, ModelTypes, relationshipToPrisma, RelationshipTypes, Searcher, selectHelper, ValidateMutationsInput } from "./base";
import { hasProfanity } from "../utils/censor";
import { CustomError } from "../error";
import _ from "lodash";

//==============================================================
/* #region Custom Components */
//==============================================================

export const resourceFormatter = (): FormatConverter<Resource> => ({}) // Needs no formatting as of now

export const resourceSearcher = (): Searcher<ResourceSearchInput> => ({
    defaultSort: ResourceSortBy.AlphabeticalAsc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceSortBy.AlphabeticalAsc]: { title: 'asc' },
            [ResourceSortBy.AlphabeticalDesc]: { title: 'desc' },
            [ResourceSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { title: { ...insensitive } },
                { description: { ...insensitive } },
                { link: { ...insensitive } },
            ]
        })
    },
    customQueries(input: ResourceSearchInput): { [x: string]: any } {
        const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        return { ...forQuery };
    },
})

// Maps resource for type to correct field
const forMap: { [x: ResourceFor]: string } = {
    [ResourceFor.Organization]: 'organizationId',
    [ResourceFor.Project]: 'projectId',
    [ResourceFor.RoutineContextual]: 'routineContextualId',
    [ResourceFor.RoutineExternal]: 'routineExternalId',
    [ResourceFor.User]: 'userId',
}

export const resourceVerifier = () => ({
    profanityCheck(data: ResourceCreateInput | ResourceUpdateInput): void {
        if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
    },
})

export const resourceMutater = (prisma: PrismaType, verifier: any) => ({
    async toDBShape(userId: string | null, data: ResourceCreateInput | ResourceUpdateInput): Promise<any> {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...rest } = data;
        // Map forId to correct field
        if (createdFor) {
            return {
                ...rest,
                organizationId: null,
                projectId: null,
                routineContextualId: null,
                routineExternalId: null,
                userId: null,
                [forMap[createdFor]]: createdForId
            };
        }
        return rest;
    },
    async relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): Promise<{ [x: string]: any } | undefined> {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'resources', isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate
        const { create: createMany, update: updateMany, delete: deleteMany } = formattedInput;
        await this.validateMutations({
            userId,
            createMany: createMany as ResourceCreateInput[],
            updateMany: updateMany as ResourceUpdateInput[],
            deleteMany: deleteMany?.map(d => d.id)
        });
        // Shape
        if (Array.isArray(formattedInput.create)) {
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
        // TODO check that user can add resource to this forId, like in node validateMutations
        if (createMany) {
            createMany.forEach(input => resourceCreate.validateSync(input, { abortEarly: false }));
            createMany.forEach(input => verifier.profanityCheck(input));
            // Check for max resources on object TODO
        }
        if (updateMany) {
            updateMany.forEach(input => resourceUpdate.validateSync(input, { abortEarly: false }));
            updateMany.forEach(input => verifier.profanityCheck(input));
        }
    },
    async cud({ info, userId, createMany, updateMany, deleteMany }: CUDInput<ResourceCreateInput, ResourceUpdateInput>): Promise<CUDResult<Resource>> {
        await this.validateMutations({ userId, createMany, updateMany, deleteMany });
        // Perform mutations
        let created: any[] = [], updated: any[] = [], deleted: Count = { count: 0 };
        if (createMany) {
            // Loop through each create input
            for (const input of createMany) {
                // Call createData helper function
                const data = await this.toDBShape(userId, input);
                // Create object
                const currCreated = await prisma.resource.create({ data, ...selectHelper(info) });
                // Convert to GraphQL
                const converted = modelToGraphQL(currCreated, info);
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
                    ...selectHelper(info)
                });
                // Convert to GraphQL
                const converted = modelToGraphQL(currUpdated, info);
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
        // Format and add supplemental/calculated fields
        const createdLength = created.length;
        const supplemental = await addSupplementalFields(prisma, userId, [...created, ...updated], info);
        created = supplemental.slice(0, createdLength);
        updated = supplemental.slice(createdLength);
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
    const model = ModelTypes.Resource;
    const prismaObject = prisma.resource;
    const format = resourceFormatter();
    const search = resourceSearcher();
    const verify = resourceVerifier();
    const mutate = resourceMutater(prisma, verify);

    return {
        model,
        prismaObject,
        ...format,
        ...search,
        ...verify,
        ...mutate,
        ...counter<ResourceCountInput>(model, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================