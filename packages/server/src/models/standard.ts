import { Routine, Standard, StandardInput, Tag, User } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type StandardRelationshipList = 'tags' | 'routineInputs' | 'routineOutputs' | 'starredBy' |
    'reports' | 'comments';
// Type 2. QueryablePrimitives
export type StandardQueryablePrimitives = Omit<Standard, StandardRelationshipList>;
// Type 3. AllPrimitives
export type StandardAllPrimitives = StandardQueryablePrimitives;
// type 4. FullModel
export type StandardFullModel = StandardAllPrimitives &
Pick<Standard, 'reports' | 'comments'> &
{
    tags: { tag: Tag[] }[],
    routineInputs: { routine: Routine[] }[],
    routineOutputs: { routine: Routine[] }[],
    starredBy: { user: User[] }[],
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

export function StandardModel(prisma?: any) {
    let obj: BaseState<Standard, StandardFullModel> = {
        prisma,
        model: MODEL_TYPES.Standard,
    }

    return {
        ...obj,
        ...findByIder<StandardFullModel>(obj),
        ...formatter(),
        ...creater<StandardInput, StandardFullModel>(obj),
        ...updater<StandardInput, StandardFullModel>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================