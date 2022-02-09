import { CODE, resourceCreate, ResourceFor, resourceUpdate } from "@local/shared";
import { Resource, ResourceCountInput, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, DeleteManyInput, Count, FindByIdInput } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { counter, FormatConverter, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipToPrisma, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { CustomError } from "../error";
import { resource } from "@prisma/client";

//==============================================================
/* #region Custom Components */
//==============================================================

// Maps resource for type to correct field
const forMap = {
    [ResourceFor.Organization]: 'organizationId',
    [ResourceFor.Project]: 'projectId',
    [ResourceFor.RoutineContextual]: 'routineContextualId',
    [ResourceFor.RoutineExternal]: 'routineExternalId',
    [ResourceFor.User]: 'userId',
}

/**
 * Component for formatting between graphql and prisma types
 */
export const resourceFormatter = (): FormatConverter<Resource, resource> => ({
    toDB: (obj: RecursivePartial<Resource>): RecursivePartial<resource> => (obj as any), //TODO
    toGraphQL: (obj: RecursivePartial<resource>): RecursivePartial<Resource> => (obj as any) //TODO
})

/**
 * Handles the authorized adding, updating, and deleting of resources.
 */
const resourcer = (format: FormatConverter<Resource, resource>, sort: Sortable<ResourceSortBy>, prisma: PrismaType) => ({
    /**
     * Add, update, or remove a resource relationship from an organization, project, routine, or user
     */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object, and 
        // it's not worth developing a way to transfer them between objects)
        let formattedInput = relationshipToPrisma(input, 'resources', isAdd, ['createdFor', 'createdForId'], false);
        delete formattedInput.connect;
        delete formattedInput.disconnect;
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const resource of formattedInput.create) {
                // Check for valid arguments
                resourceCreate.omit(['createdFor', 'createdForId']).validateSync(input, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(resource.title, resource.description)) throw new CustomError(CODE.BannedWord);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const resource of formattedInput.update) {
                // Check for valid arguments
                resourceUpdate.omit(['createdFor', 'createdForId']).validateSync(input, { abortEarly: false });
                // Check for censored words
                if (hasProfanity(resource.title, resource.description)) throw new CustomError(CODE.BannedWord);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Resource> | null> {
        // Create selector
        const select = selectHelper<Resource, resource>(info, format.toDB);
        // Access database
        const resource = await prisma.resource.findUnique({ where: { id: input.id }, ...select });
        return resource ? format.toGraphQL(resource) : null;
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: ResourceSearchInput,
        info: InfoType = null,
    ): Promise<PaginatedSearchResult> {
        const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        // Search
        const search = searcher<ResourceSortBy, ResourceSearchInput, Resource, resource>(MODEL_TYPES.Resource, format.toDB, format.toGraphQL, sort, prisma);
        let searchResults = await search.search({ ...forQuery, ...where }, input, info);
        return searchResults;
    },
    async create(
        userId: string,
        input: ResourceCreateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Resource>> {
        // TODO authorize user
        // Check for valid arguments
        resourceCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.title, input.description)) throw new CustomError(CODE.BannedWord);
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...resourceData } = input;
        // Map forId to correct field
        const data = { ...resourceData, [forMap[createdFor]]: createdForId }
        // Create
        const resource = await prisma.resource.create({ data: data as any });
        // Return query
        return await prisma.resource.findFirst({ where: { id: resource.id }, ...selectHelper<Resource, resource>(info, format.toDB) }) as any;
    },
    async update(
        userId: string,
        input: ResourceUpdateInput,
        info: InfoType = null,
    ): Promise<RecursivePartial<Resource>> {
        // TODO authorize user
        // Check for valid arguments
        resourceUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        if (hasProfanity(input.title, input.description)) throw new CustomError(CODE.BannedWord);
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...resourceData } = input;
        // Map forId to correct field, and remove any old associations
        const data = (Boolean(createdFor) && Boolean(createdForId)) ? {
            ...resourceData,
            organizationId: null,
            projectId: null,
            routineContextualId: null,
            routineExternalId: null,
            userId: null,
            [forMap[createdFor as any]]: createdForId
        } : resourceData;
        // Update resource
        return await prisma.resource.update({
            where: { id: input.id },
            data,
            ...selectHelper<Resource, resource>(info, format.toDB)
        }) as any;
    },
    async delete(userId: string, input: DeleteManyInput): Promise<Count> {
        // TODO authorize user
        // Delete
        return await prisma.resource.deleteMany({
            where: { id: { in: input.ids } },
        });
    }
})

/**
 * Component for search filters
 */
export const resourceSorter = (): Sortable<ResourceSortBy> => ({
    defaultSort: ResourceSortBy.AlphabeticalDesc,
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
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ResourceModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Resource;
    const format = resourceFormatter();
    const sort = resourceSorter();

    return {
        model,
        prisma,
        ...format,
        ...sort,
        ...counter<ResourceCountInput>(model, prisma),
        ...resourcer(format, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================