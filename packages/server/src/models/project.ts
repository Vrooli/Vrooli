import { Organization, Project, ProjectInput, Resource, Tag, User } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// // Type 1. RelationshipList
// export type ProjectRelationshipList = 'resources' | 'wallets' | 'users' | 'organizations' | 'starredBy' | 
//     'parent' | 'forks' | 'reports' | 'tags' | 'comments';
// // Type 2. QueryablePrimitives
// export type ProjectQueryablePrimitives = Omit<Project, ProjectRelationshipList>;
// // Type 3. AllPrimitives
// export type ProjectAllPrimitives = ProjectQueryablePrimitives;
// // type 4. FullModel
// export type ProjectFullModel = ProjectAllPrimitives &
// Pick<Project, 'wallets' | 'reports' | 'comments'> &
// {
//     resources: { resource: Resource[] }[],
//     users: { user: User[] }[],
//     organizations: { organization: Organization[] }[],
//     starredBy: { user: User[] }[],
//     parent: { project: Project[] }[],
//     forks: { project: Project[] }[],
//     tags: { tag: Tag[] }[],
// };

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

export function ProjectModel(prisma: any) {
    let obj: BaseState<Project> = {
        prisma,
        model: MODEL_TYPES.Project,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Project>(obj),
        ...formatter(),
        ...creater<ProjectInput, Project>(obj),
        ...updater<ProjectInput, Project>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================