import { Comment, CommentInput, User } from "schema/types";
import { RecursivePartial } from "types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type CommentRelationshipList = 'user' | 'organization' | 'project' | 'resource' |
    'routine' | 'standard' | 'reports' | 'stars' | 'votes';
// Type 2. QueryablePrimitives
export type CommentQueryablePrimitives = Omit<Comment, CommentRelationshipList>;
// Type 3. AllPrimitives
export type CommentAllPrimitives = CommentQueryablePrimitives;
// type 4. FullModel
export type CommentFullModel = CommentAllPrimitives &
    Pick<Comment, 'user' | 'organization' | 'project' | 'resource' | 'routine' | 'standard' | 'reports'> &
{
    stars: number,
    votes: number,
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
const formatter = (): FormatConverter<any, any> => ({
    toDB: (obj: RecursivePartial<Comment>): RecursivePartial<any> => obj as any,
    toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Comment> => obj as any
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma: any) {
    let obj: BaseState<Comment> = {
        prisma,
        model: MODEL_TYPES.Comment,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Comment>(obj),
        ...formatter(),
        ...creater<CommentInput, Comment>(obj),
        ...updater<CommentInput, Comment>(obj),
        ...deleter(obj),
        ...reporter()
    }
}

//==============================================================
/* #endregion Model */
//==============================================================