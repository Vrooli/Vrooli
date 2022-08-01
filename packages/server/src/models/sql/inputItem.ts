import { InputItem } from "../../schema/types";
import { PrismaType } from "../../types";
import { FormatConverter } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const inputItemFormatter = (): FormatConverter<InputItem> => ({
    relationshipMap: {
        '__typename': 'InputItem',
        'standard': 'Standard',
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const InputItemModel = ({
    prismaObject: (prisma: PrismaType) => prisma.routine_input,
    format: inputItemFormatter(),
})

//==============================================================
/* #endregion Model */
//==============================================================