import { OutputItem } from "../../schema/types";
import { PrismaType } from "types";
import { FormatConverter, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const outputItemFormatter = (): FormatConverter<OutputItem> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.OutputItem,
        'standard': GraphQLModelType.Standard,
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function OutputItemModel(prisma: PrismaType) {
    const prismaObject = prisma.routine_output;
    const format = outputItemFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
    }
}
//==============================================================
/* #endregion Model */
//==============================================================