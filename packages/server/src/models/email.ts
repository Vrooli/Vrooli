import { Email, EmailInput } from "schema/types";
import { PrismaType } from "types";
import { creater, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type EmailRelationshipList = 'user';
// Type 2. QueryablePrimitives
export type EmailQueryablePrimitives = Omit<Email, EmailRelationshipList>;
// Type 3. AllPrimitives
export type EmailAllPrimitives = EmailQueryablePrimitives & {
    verificationCode: string | null;
    lastVerificationCodeRequestAttempt: Date | null;
}
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

export function EmailModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Email;
    
    return {
        prisma,
        model,
        ...findByIder<EmailFullModel>(model, prisma),
        ...formatter(),
        ...creater<EmailInput, EmailFullModel>(model, prisma),
        ...updater<EmailInput, EmailFullModel>(model, prisma),
        ...deleter(model, prisma)
    }
}

//==============================================================
/* #endregion Model */
//==============================================================