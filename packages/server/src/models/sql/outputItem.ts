import { OutputItem } from "../../schema/types";
import { PrismaType } from "../../types";
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

export const OutputItemModel = ({
    prismaObject: (prisma: PrismaType) => prisma.routine_output,
    format: outputItemFormatter(),
})

//==============================================================
/* #endregion Model */
//==============================================================