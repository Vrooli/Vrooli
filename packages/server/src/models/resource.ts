import { resourceAdd, ResourceFor, resourceUpdate } from "@local/shared";
import { PrismaSelect } from "@paljs/plugins";
import { Organization, Project, Resource, ResourceCountInput, ResourceAddInput, ResourceUpdateInput, ResourceSearchInput, ResourceSortBy, Routine, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { counter, deleter, findByIder, FormatConverter, MODEL_TYPES, searcher, Sortable } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type ResourceRelationshipList = 'organization_resources' | 'project_resources' | 'routine_resources_contextual' |
    'routine_resources_external' | 'user_resources';
// Type 2. QueryablePrimitives
export type ResourceQueryablePrimitives = Omit<Resource, ResourceRelationshipList>;
// Type 3. AllPrimitives
export type ResourceAllPrimitives = ResourceQueryablePrimitives;
// type 4. Database shape
export type ResourceDB = ResourceAllPrimitives &
{
    organization_resources: { organization: Organization },
    project_resources: { project: Project },
    routine_resources_contextual: { routine: Routine },
    routine_resources_external: { routine: Routine },
    user_resources: { user: User },
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

// Maps resource for type to correct fiel
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
export const resourceFormatter = (): FormatConverter<Resource, ResourceDB> => ({
    toDB: (obj: RecursivePartial<Resource>): RecursivePartial<ResourceDB> => (obj as any), //TODO
    toGraphQL: (obj: RecursivePartial<ResourceDB>): RecursivePartial<Resource> => (obj as any) //TODO
})

/**
 * Custom compositional component for creating resources
 * @param state 
 * @returns 
 */
export const resourceCreater = (prisma: PrismaType) => ({
    async create(data: ResourceAddInput, info: any): Promise<RecursivePartial<ResourceDB>> {
        // TODO authorize user
        // Check for valid arguments
        resourceAdd.validateSync(data, { abortEarly: false });
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...resourceData } = data;
        // Map forId to correct field
        const createData = { ...resourceData, [forMap[createdFor]]: createdForId }
        // Create
        const resource = await prisma.resource.create({ data: createData as any });
        // Return query
        return await prisma.resource.findFirst({ where: { id: resource.id }, ...(new PrismaSelect(info).value) }) as any;
    }
})

/**
 * Custom compositional component for updating resources
 * @param state 
 * @returns 
 */
const resourceUpdater = (prisma: PrismaType) => ({
    async update(data: ResourceUpdateInput, info: any): Promise<ResourceDB> {
        // TODO authorize user
        // Check for valid arguments
        resourceUpdate.validateSync(data, { abortEarly: false });
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, createdForId, ...resourceData } = data;
        // Map forId to correct field, and remove any old associations
        const updateData = (Boolean(createdFor) && Boolean(createdForId)) ? { 
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
            where: { id: data.id },
            data: updateData,
            ...(new PrismaSelect(info).value)
        }) as any;
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
        ...deleter(model, prisma),
        ...findByIder<Resource, ResourceDB>(model, format.toDB, prisma),
        ...resourceCreater(prisma),
        ...resourceUpdater(prisma),
        ...searcher<ResourceSortBy, ResourceSearchInput, Resource, ResourceDB>(model, format.toDB, format.toGraphQL, sort, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================