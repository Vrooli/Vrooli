import { Email, EmailInput } from "schema/types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type EmailRelationshipList = 'user';
// Type 2. QueryablePrimitives
export type EmailQueryablePrimitives = Omit<Email, EmailRelationshipList>;
// Type 3. AllPrimitives
export type EmailAllPrimitives = EmailQueryablePrimitives;
// type 4. FullModel
export type EmailFullModel = EmailAllPrimitives &
Pick<Email, 'user'>;

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

export function EmailModel(prisma?: any) {
    let obj: BaseState<Email, EmailFullModel> = {
        prisma,
        model: MODEL_TYPES.Email,
    }
    
    return {
        ...obj,
        ...findByIder<EmailFullModel>(obj),
        ...formatter(),
        ...creater<EmailInput, EmailFullModel>(obj),
        ...updater<EmailInput, EmailFullModel>(obj),
        ...deleter(obj)
    }
}

//==============================================================
/* #endregion Model */
//==============================================================