import { OutputItem } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, GraphQLModelType } from "./types";

//==============================================================
/* #region Custom Components */
//==============================================================

export const outputItemFormatter = (): FormatConverter<OutputItem, any> => ({
    relationshipMap: {
        '__typename': 'OutputItem',
        'standard': 'Standard',
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const OutputItemModel = ({
    prismaObject: (prisma: PrismaType) => prisma.routine_version_output,
    format: outputItemFormatter(),
    type: 'OutputItem' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================