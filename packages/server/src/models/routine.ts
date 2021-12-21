import { Organization, Resource, Routine, RoutineInput, Standard, Tag, User } from "schema/types";
import { RecursivePartial } from "types";
import { addJoinTables, BaseState, creater, deleter, FormatConverter, MODEL_TYPES, removeJoinTables, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type RoutineRelationshipList = 'inputs' | 'outputs' | 'nodes' | 'contextualResources' |
    'externalResources' | 'donationResources' | 'tags' | 'users' | 'organizations' | 'starredBy' | 
    'parent' | 'forks' | 'nodeLists' | 'reports' | 'comments';
// Type 2. QueryablePrimitives
export type RoutineQueryablePrimitives = Omit<Routine, RoutineRelationshipList>;
// Type 3. AllPrimitives
export type RoutineAllPrimitives = RoutineQueryablePrimitives;
// type 4. FullModel
export type RoutineFullModel = RoutineAllPrimitives &
Pick<Routine, 'nodes' | 'reports' | 'comments' | 'inputs' | 'outputs' | 'parent'> &
{
    contextualResources: { resource: Resource[] }[],
    externalResources: { resource: Resource[] }[],
    donationResources: { resource: Resource[] }[],
    tags: { tag: Tag[] }[],
    users: { user: User[] }[],
    organizations: { organization: Organization[] }[],
    starredBy: { user: User[] }[],
    forks: { fork: Routine[] }[],
    nodeLists: { list: Routine[] }[],
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
 const formatter = (): FormatConverter<Routine, any> => {
    const joinMapper = {
        contextualResources: 'resource',
        externalResources: 'resource',
        donationResources: 'resource',
        tags: 'tag',
        users: 'user',
        organizations: 'organization',
        starredBy: 'user',
        forks: 'fork',
        nodeLists: 'list',
    };
    return {
        toDB: (obj: RecursivePartial<Routine>): RecursivePartial<any> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Routine> => removeJoinTables(obj, joinMapper)
    }
}

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoutineModel(prisma?: any) {
    let obj: BaseState<Routine> = {
        prisma,
        model: MODEL_TYPES.Routine,
        format: formatter(),
    }

    return {
        ...obj,
        ...creater<RoutineInput, Routine>(obj),
        ...formatter(),
        ...updater<RoutineInput, Routine>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================