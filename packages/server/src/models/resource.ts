import { PrismaSelect } from "@paljs/plugins";
import { Organization, Project, Resource, ResourceInput, Routine, User } from "schema/types";
import { BaseState, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// // Type 1. RelationshipList
// export type ResourceRelationshipList = 'organization_resources' | 'project_resources' | 'routine_resources_contextual' | 
// 'routine_resources_external' | 'routine_resources_donation' | 'user_resources' | 'starredBy' | 'reports' | 'comments';
// // Type 2. QueryablePrimitives
// export type ResourceQueryablePrimitives = Omit<Resource, ResourceRelationshipList>;
// // Type 3. AllPrimitives
// export type ResourceAllPrimitives = ResourceQueryablePrimitives;
// // type 4. FullModel
// export type ResourceFullModel = ResourceAllPrimitives &
// Pick<Resource, 'reports' | 'comments'> &
// {
//     organization_resources: { organization: Organization[] },
//     project_resources: { project: Project[] },
//     routine_resources_contextual: { routine: Routine[] },
//     routine_resources_external: { routine: Routine[] },
//     routine_resources_donation: { routine: Routine[] },
//     user_resources: { user: User[] },
//     starredBy: { user: User[] }[],
// };

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

// Maps routine apply types to the correct prisma join tables
const applyMap = {
    ORGANIZATION: 'organizationResources',
    PROJECT: 'projectResources',
    ROUTINE_CONTEXTUAL: 'routineResourcesContextual',
    ROUTINE_DONATION: 'routineResourcesDonation',
    ROUTINE_EXTERNAL: 'routineResourcesExternal',
    USER: 'userResources'
}

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

/**
 * Custom compositional component for creating resources
 * @param state 
 * @returns 
 */
const creater = (state: any) => ({
    /**
    * Applies a resource object to an actor, project, organization, or routine
    * @param resource 
    * @returns
    */
    async applyToObject(resource: any, createdFor: keyof typeof applyMap, forId: string): Promise<any> {
        return await state.prisma[applyMap[createdFor]].create({
            data: {
                forId,
                resourceId: resource.id
            }
        })
    },
    async create(data: ResourceInput, info: any): Promise<Resource> {
         // Filter out for and forId, since they are not part of the resource object
         const { createdFor, forId, ...resourceData } = data;
         // Create base object
         const resource = await state.prisma.resouce.create({ data: resourceData });
         // Create join object
         await this.applyToObject(resource, createdFor, forId);
         // Return query
         return await state.prisma.resource.findOne({ where: { id: resource.id }, ...(new PrismaSelect(info).value) });
    }
})

/**
 * Custom compositional component for updating resources
 * @param state 
 * @returns 
 */
const updater = (state: any) => ({
    async update(data: ResourceInput, info: any): Promise<Resource> {
        // Filter out for and forId, since they are not part of the resource object
        const { createdFor, forId, ...resourceData } = data;
        // Check if resource needs to be associated with another object instead
        //TODO
        // Update base object and return query
        return await state.prisma.resource.update({
            where: { id: data.id },
            data: resourceData,
            ...(new PrismaSelect(info).value)
        });
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function ResourceModel(prisma: any) {
    let obj: BaseState<Resource> = {
        prisma,
        model: MODEL_TYPES.Resource,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Resource>(obj),
        ...formatter(),
        ...creater(obj),
        ...updater(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================