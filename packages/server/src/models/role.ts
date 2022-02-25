import { role } from "@prisma/client";
import _ from "lodash";
import { RecursivePartial, PrismaType, PartialSelectConvert } from "types";
import { Role } from "../schema/types";
import { FormatConverter, addJoinTables, removeJoinTables, MODEL_TYPES, InfoType, infoToPartialSelect } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

type RoleFormatterType = FormatConverter<Role, role>;
/**
 * Component for formatting between graphql and prisma types
 */
export const roleFormatter = (): RoleFormatterType => {
    const joinMapper = {
        users: 'user',
    };
    return {
        dbShape: (partial: PartialSelectConvert<Role>): PartialSelectConvert<role> => {
            let modified = partial;
            // Add join tables not present in GraphQL type, but present in Prisma
            modified = addJoinTables(modified, joinMapper);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<role> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<role> => {
            return roleFormatter().dbShape(roleFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<role>): RecursivePartial<Role> => {
            // Remove join tables that are not present in GraphQL type, but present in Prisma
            let modified = removeJoinTables(obj, joinMapper)
            return modified;
        }
    }
}

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function RoleModel(prisma: PrismaType) {
    const model = MODEL_TYPES.Role;
    const format = roleFormatter();

    return {
        prisma,
        model,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================