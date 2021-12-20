import { Standard, StandardInput } from "schema/types";
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

export function StandardModel(prisma: any) {
    let obj: BaseState<Standard> = {
        prisma,
        model: MODEL_TYPES.Standard,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Standard>(obj),
        ...formatter(),
        ...creater<StandardInput, Standard>(obj),
        ...updater<StandardInput, Standard>(obj),
        ...deleter(obj),
        ...reporter()
    }
}