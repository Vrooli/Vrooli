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
// type 4. Database shape
export type EmailDB = EmailAllPrimitives &
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
 export const emailFormatter = (): FormatConverter<any, any>  => ({
    toDB: (obj: any): any => ({ ...obj}),
    toGraphQL: (obj: any): any => ({ ...obj })
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function EmailModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Email;
    const format = emailFormatter();
    
    return {
        prisma,
        model,
        ...format,
        ...findByIder<Email, EmailDB>(model, format.toDB, prisma),
        ...creater<EmailInput, Email, EmailDB>(model, format.toDB, prisma),
        ...updater<EmailInput, Email, EmailDB>(model, format.toDB, prisma),
        ...deleter(model, prisma)
    }
}

//==============================================================
/* #endregion Model */
//==============================================================