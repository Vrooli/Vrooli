import { PrismaType } from "../types";
import { FormatConverter } from "./base";
import { GraphQLModelType } from ".";
import { Member } from "../schema/types";

//==============================================================
/* #region Custom Components */
//==============================================================

export const memberFormatter = (): FormatConverter<Member, any> => ({
    relationshipMap: {
        '__typename': 'Member',
        'organization': 'Organization',
        'user': 'User',
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const MemberModel = ({
    prismaObject: (prisma: PrismaType) => prisma.organization_users,
    format: memberFormatter(),
    type: 'Member' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================