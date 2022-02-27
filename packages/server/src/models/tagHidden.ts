import { PrismaType } from "types";
import { TagHidden } from "../schema/types";
import { FormatConverter, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const tagHiddenFormatter = (): FormatConverter<TagHidden> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.TagHidden,
        'user': GraphQLModelType.User,
        'organization': GraphQLModelType.Organization,
        'project': GraphQLModelType.Project,
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function TagHiddenModel(prisma: PrismaType) {
    const prismaObject = prisma.wallet;
    const format = tagHiddenFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
    }
}

//==============================================================
/* #endregion Model */
//==============================================================