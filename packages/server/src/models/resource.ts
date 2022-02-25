import { CODE, resourceCreate, ResourceFor, resourceUpdate } from "@local/shared";
import { Resource, ResourceCountInput, ResourceCreateInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, DeleteManyInput, Count, FindByIdInput } from "../schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "types";
import { counter, FormatConverter, infoToPartialSelect, InfoType, MODEL_TYPES, PaginatedSearchResult, relationshipToPrisma, RelationshipTypes, searcher, selectHelper, Sortable } from "./base";
import { hasProfanity } from "../utils/censor";
import { CustomError } from "../error";
import { resource } from "@prisma/client";
import _ from "lodash";

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

type ResourceFormatterType = FormatConverter<Resource, resource>;
/**
 * Component for formatting between graphql and prisma types
 */
export const resourceFormatter = (): ResourceFormatterType => ({
    dbShape: (partial: PartialSelectConvert<Resource>): PartialSelectConvert<resource> => {
        let modified = partial;
        return modified;
    },
    dbPrune: (info: InfoType): PartialSelectConvert<resource> => {
        // Convert GraphQL info object to a partial select object
        let modified = infoToPartialSelect(info);
        return modified;
    },
    selectToDB: (info: InfoType): PartialSelectConvert<resource> => {
        return resourceFormatter().dbShape(resourceFormatter().dbPrune(info));
    },
    selectToGraphQL: (obj: RecursivePartial<resource>): RecursivePartial<Resource> => {
        return obj as any;
    },
})

/**
 * Handles the authorized adding, updating, and deleting of resources.
 */
const resourcer = (format: ResourceFormatterType, sort: Sortable<ResourceSortBy>, prisma: PrismaType) => ({
    /**
     * Add, update, or remove a resource relationship from an organization, project, routine, or user
     */
    relationshipBuilder(
        userId: string | null,
        input: { [x: string]: any },
        isAdd: boolean = true,
    ): { [x: string]: any } | undefined {
        const fieldExcludes = ['createdFor', 'createdForId'];
        // Convert input to Prisma shape, excluding "createdFor" and "createdForId" fields
        // Also remove anything that's not an create, update, or delete, as connect/disconnect
        // are not supported by resources (since they can only be applied to one object, and 
        // it's not worth developing a way to transfer them between objects)
        let formattedInput = relationshipToPrisma({ data: input, relationshipName: 'resources', isAdd, fieldExcludes, relExcludes: [RelationshipTypes.connect, RelationshipTypes.disconnect] })
        // Validate create
        if (Array.isArray(formattedInput.create)) {
            for (const resource of formattedInput.create) {
                // Check for valid arguments
                resourceCreate.omit(fieldExcludes).validateSync(resource, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(resource as ResourceCreateInput);
            }
        }
        // Validate update
        if (Array.isArray(formattedInput.update)) {
            for (const resource of formattedInput.update) {
                // Check for valid arguments
                resourceUpdate.omit(fieldExcludes).validateSync(resource, { abortEarly: false });
                // Check for censored words
                this.profanityCheck(resource as ResourceUpdateInput);
            }
        }
        return Object.keys(formattedInput).length > 0 ? formattedInput : undefined;
    },
    async find(
        userId: string | null | undefined, // Of the user making the request, not the organization
        input: FindByIdInput,
        info: InfoType,
    ): Promise<RecursivePartial<Resource> | null> {
        // Create selector
        const select = selectHelper(info, format.selectToDB);
        // Access database
        const resource = await prisma.resource.findUnique({ where: { id: input.id }, ...select });
        return resource ? format.selectToGraphQL(resource) : null;
    },
    async search(
        where: { [x: string]: any },
        userId: string | null | undefined,
        input: ResourceSearchInput,
        info: InfoType,
    ): Promise<PaginatedSearchResult> {
        const forQuery = (input.forId && input.forType) ? { [forMap[input.forType]]: input.forId } : {};
        // Search
        const search = searcher<ResourceSortBy, ResourceSearchInput, Resource, resource>(MODEL_TYPES.Resource, format.selectToDB, sort, prisma);
        console.log('calling resource search...')
        let searchResults = await search.search({ ...forQuery, ...where }, input, info);
        return searchResults;
    },
    async create(
        userId: string,
        input: ResourceCreateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Resource>> {
        // TODO authorize user
        // Check for valid arguments
        resourceCreate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...resourceData } = input;
        // Map forId to correct field
        const data = { ...resourceData, [forMap[createdFor]]: createdForId as any }
        // Create
        const resource = await prisma.resource.create({
            data: data,
            ...selectHelper(info, format.selectToDB)
        });
        return { ...format.selectToGraphQL(resource) }
    },
    async update(
        userId: string,
        input: ResourceUpdateInput,
        info: InfoType,
    ): Promise<RecursivePartial<Resource>> {
        // TODO authorize user
        // Check for valid arguments
        resourceUpdate.validateSync(input, { abortEarly: false });
        // Check for censored words
        this.profanityCheck(input);
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
            ...selectHelper(info, format.selectToDB)
        }) as any;
    },
    async delete(userId: string, input: DeleteManyInput): Promise<Count> {
        // TODO authorize user
        // Delete
        return await prisma.resource.deleteMany({
            where: { id: { in: input.ids } },
        });
    },
    profanityCheck(data: ResourceCreateInput | ResourceUpdateInput): void {
        if (hasProfanity(data.title, data.description)) throw new CustomError(CODE.BannedWord);
    },
})

/**
 * Component for search filters
 */
export const resourceSorter = (): Sortable<ResourceSortBy> => ({
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