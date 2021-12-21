import { Organization, OrganizationInput, Project, Resource, Routine, Tag, User } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type OrganizationRelationshipList = 'comments' | 'resources' | 'wallets' | 'projects' | 'starredBy' | 
    'routines' | 'tags' | 'reports';
// Type 2. QueryablePrimitives
export type OrganizationQueryablePrimitives = Omit<Organization, OrganizationRelationshipList>;
// Type 3. AllPrimitives
export type OrganizationAllPrimitives = OrganizationQueryablePrimitives;
// type 4. FullModel
export type OrganizationFullModel = OrganizationAllPrimitives &
Pick<Organization, 'comments' | 'wallets' | 'reports'> &
{
    resources: { resource: Resource[] }[],
    projects: { project: Project[] }[],
    starredBy: { user: User[] }[],
    routines: { routine: Routine[] }[],
    tags: { tag: Tag[] }[],
};

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function OrganizationModel(prisma: any) {
    let obj: BaseState<Organization> = {
        prisma,
        model: MODEL_TYPES.Organization,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Organization>(obj),
        ...formatter(),
        ...creater<OrganizationInput, Organization>(obj),
        ...updater<OrganizationInput, Organization>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================