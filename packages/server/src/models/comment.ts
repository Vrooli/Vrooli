import { Comment, CommentInput } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

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