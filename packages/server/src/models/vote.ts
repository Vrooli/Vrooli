import { Vote, VoteInput } from "schema/types";
import { PrismaType, RecursivePartial } from "types";
import { creater, deleter, findByIder, FormatConverter, MODEL_TYPES, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type VoteRelationshipList = 'comment' | 'project' | 'routine' | 'standard' | 'tag' | 'user';
// Type 2. QueryablePrimitives
export type VoteQueryablePrimitives = Omit<Vote, VoteRelationshipList>;
// Type 3. AllPrimitives
export type VoteAllPrimitives = VoteQueryablePrimitives;
// type 4. Database shape
export type VoteDB = any;
// export type VoteDB = VoteAllPrimitives &
//     Pick<Vote, 'comment' | 'project' | 'routine' | 'standard' | 'tag' | 'user'>

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
    toDB: (obj: RecursivePartial<Vote>): RecursivePartial<any> => obj as any,
    toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Vote> => obj as any
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function VoteModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Vote;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...creater<VoteInput, Vote, VoteDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Vote, VoteDB>(model, format.toDB, prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================