import { Email, EmailInput } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";

//======================================================================================================================
// START Type definitions
//======================================================================================================================

//======================================================================================================================
// END Type definitions
//======================================================================================================================

/**
 * Component for formatting between graphql and prisma types
 */
 const formatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

export function EmailModel(prisma: any) {
    let obj: BaseState<Email> = {
        prisma,
        model: MODEL_TYPES.Email,
        format: formatter(),
    }
    
    return {
        ...obj,
        ...findByIder<Email>(obj),
        ...formatter(),
        ...creater<EmailInput, Email>(obj),
        ...updater<EmailInput, Email>(obj),
        ...deleter(obj)
    }
}