import { OutputItem } from "../../schema/types";
import { PrismaType } from "../../types";
import { FormatConverter } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const outputItemFormatter = (): FormatConverter<OutputItem> => ({
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
    prismaObject: (prisma: PrismaType) => prisma.routine_output,
    format: outputItemFormatter(),
})

//==============================================================
/* #endregion Model */
//==============================================================