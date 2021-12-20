import { Tag, TagInput } from "schema/types";
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

export function TagModel(prisma: any) {
    let obj: BaseState<Tag> = {
        prisma,
        model: MODEL_TYPES.Tag,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Tag>(obj),
        ...creater<TagInput, Tag>(obj),
        ...formatter(),
        ...updater<TagInput, Tag>(obj),
        ...deleter(obj),
        ...reporter()
    }
}