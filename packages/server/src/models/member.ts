import { PrismaType } from "../types";
import { Member } from "../schema/types";
import { FormatConverter } from "./builder";
import { GraphQLModelType } from ".";

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