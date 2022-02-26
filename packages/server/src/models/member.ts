import { PrismaType } from "../types";
import { Member } from "../schema/types";
import { FormatConverter } from "./base";

export const memberDBFields = ['id', 'role'];

//==============================================================
/* #region Custom Components */
//==============================================================

export const memberFormatter = (): FormatConverter<Member> => ({ }) // No formatting required so far

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function MemberModel(prisma: PrismaType) {
    const prismaObject = prisma.organization_users;
    const format = memberFormatter();

    return {
        prismaObject,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================