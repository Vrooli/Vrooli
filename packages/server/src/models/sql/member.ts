import { PrismaType } from "../../types";
import { Member } from "../../schema/types";
import { FormatConverter, GraphQLModelType, ModelLogic } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const memberFormatter = (): FormatConverter<Member> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Member,
        'organization': GraphQLModelType.Organization,
        'user': GraphQLModelType.User,
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
})

//==============================================================
/* #endregion Model */
//==============================================================