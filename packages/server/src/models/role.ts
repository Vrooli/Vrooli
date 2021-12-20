import { RecursivePartial, PrismaType } from "types";
import { Role, User } from "../schema/types";
import { FormatConverter, addJoinTables, removeJoinTables, BaseState, MODEL_TYPES, findByIder } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type RoleRelationshipList = 'users';
// Type 2. QueryablePrimitives
export type RoleQueryablePrimitives = Omit<Role, RoleRelationshipList>;
// Type 3. AllPrimitives
export type RoleAllPrimitives = RoleQueryablePrimitives;
// type 4. FullModel
export type RoleFullModel = RoleAllPrimitives &
{
    users: { user: User[] }[],
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
 const formatter = (): FormatConverter<Role, RoleFullModel> => {
    const joinMapper = {
        users: 'user',
    };
    return {
        toDB: (obj: RecursivePartial<Role>): RecursivePartial<RoleFullModel> => addJoinTables(obj, joinMapper),
        toGraphQL: (obj: RecursivePartial<RoleFullModel>): RecursivePartial<Role> => removeJoinTables(obj, joinMapper)
    }
}

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoleModel(prisma?: PrismaType) {
    let obj: BaseState<Role> = {
        prisma,
        model: MODEL_TYPES.Role,
        format: formatter(),
    }

    return {
        ...obj,
        ...findByIder<Role>(obj),
        ...formatter(),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================