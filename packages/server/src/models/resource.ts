import { CODE, ResourceFor } from "@local/shared";
import { PrismaSelect } from "@paljs/plugins";
import { Organization, Project, Resource, ResourceCountInput, ResourceInput, ResourceSearchInput, ResourceSortBy, Routine, User } from "../schema/types";
import { PrismaType, RecursivePartial } from "types";
import { counter, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, searcher, Sortable } from "./base";
import { CustomError } from "../error";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type ResourceRelationshipList = 'organization_resources' | 'project_resources' | 'routine_resources_contextual' |
    'routine_resources_external' | 'routine_resources_donation' | 'user_resources' | 'starredBy' | 'reports' | 'comments';
// Type 2. QueryablePrimitives
export type ResourceQueryablePrimitives = Omit<Resource, ResourceRelationshipList>;
// Type 3. AllPrimitives
export type ResourceAllPrimitives = ResourceQueryablePrimitives;
// type 4. FullModel
export type ResourceFullModel = ResourceAllPrimitives &
    Pick<Resource, 'reports' | 'comments'> &
{
    organization_resources: { organization: Organization[] },
    project_resources: { project: Project[] },
    routine_resources_contextual: { routine: Routine[] },
    routine_resources_external: { routine: Routine[] },
    routine_resources_donation: { routine: Routine[] },
    user_resources: { user: User[] },
    starredBy: { user: User[] }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

// Maps routine apply types to the correct prisma join tables
const applyMap = {
    [ResourceFor.Actor]: 'userResources',
    [ResourceFor.Organization]: 'organizationResources',
    [ResourceFor.Project]: 'projectResources',
    [ResourceFor.RoutineContextual]: 'routineResourcesContextual',
    [ResourceFor.RoutineDonation]: 'routineResourcesDonation',
    [ResourceFor.RoutineExternal]: 'routineResourcesExternal',
}

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<Resource, ResourceFullModel> => ({
    toDB: (obj: RecursivePartial<Resource>): RecursivePartial<ResourceFullModel> => (obj as any), //TODO
    toGraphQL: (obj: RecursivePartial<ResourceFullModel>): RecursivePartial<Resource> => (obj as any) //TODO
})
/**
 * Custom compositional component for creating resources
 * @param state 
 * @returns 
 */
const creater = (prisma?: PrismaType) => ({
    /**
    * Applies a resource object to an actor, project, organization, or routine
    * @param resource 
    * @returns
    */
    async applyToObject(resource: any, createdFor: keyof typeof applyMap, forId: string): Promise<any> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        return await (prisma[applyMap[createdFor] as keyof PrismaType] as any).create({
            data: {
                forId,
                resourceId: resource.id
            }
        })
    },
    async create(data: ResourceInput, info: any): Promise<RecursivePartial<ResourceFullModel>> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Create base object
        const resource = await prisma.resource.create({ data: resourceData as any });
        // Create join object
        await this.applyToObject(resource, createdFor, forId);
        // Return query
        return await prisma.resource.findFirst({ where: { id: resource.id }, ...(new PrismaSelect(info).value) }) as any;
    }
})

/**
 * Custom compositional component for updating resources
 * @param state 
 * @returns 
 */
const updater = (prisma?: PrismaType) => ({
    async update(data: ResourceInput, info: any): Promise<ResourceFullModel> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Check if resource needs to be associated with another object instead
        //TODO
        // Update base object and return query
        return await prisma.resource.update({
            where: { id: data.id },
            data: resourceData,
            ...(new PrismaSelect(info).value)
        }) as any;
    }
})

/**
 * Component for search filters
 */
const sorter = (): Sortable<ResourceSortBy> => ({
    defaultSort: ResourceSortBy.AlphabeticalDesc,
    getSortQuery: (sortBy: string): any => {
        return {
            [ResourceSortBy.AlphabeticalAsc]: { title: 'asc' },
            [ResourceSortBy.AlphabeticalDesc]: { title: 'desc' },
            [ResourceSortBy.CommentsAsc]: { comments: { count: 'asc' } },
            [ResourceSortBy.CommentsDesc]: { comments: { count: 'desc' } },
            [ResourceSortBy.DateCreatedAsc]: { created_at: 'asc' },
            [ResourceSortBy.DateCreatedDesc]: { created_at: 'desc' },
            [ResourceSortBy.DateUpdatedAsc]: { updated_at: 'asc' },
            [ResourceSortBy.DateUpdatedDesc]: { updated_at: 'desc' },
            [ResourceSortBy.StarsAsc]: { stars: { count: 'asc' } },
            [ResourceSortBy.StarsDesc]: { stars: { count: 'desc' } },
        }[sortBy]
    },
    getSearchStringQuery: (searchString: string): any => {
        const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' });
        return ({
            OR: [
                { title: { ...insensitive } },
                { description: { ...insensitive } },
                { link: { ...insensitive } },
                { displayUrl: { ...insensitive } },
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

export function ResourceModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Resource;
    const format = formatter();
    const sort = sorter();

    return {
        model,
        prisma,
        ...format,
        ...sort,
        ...counter<ResourceCountInput>(model, prisma),
        ...creater(prisma),
        ...deleter(model, prisma),
        ...findByIder<ResourceFullModel>(model, prisma),
        ...reporter(),
        ...searcher<ResourceSortBy, ResourceSearchInput, Resource, ResourceFullModel>(model, format.toGraphQL, sort, prisma),
        ...updater(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================