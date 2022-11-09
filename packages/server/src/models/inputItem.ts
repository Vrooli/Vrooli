import { InputItem } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, GraphQLModelType } from "./types";

//==============================================================
/* #region Custom Components */
//==============================================================

export const inputItemFormatter = (): FormatConverter<InputItem, any> => ({
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
    prismaObject: (prisma: PrismaType) => prisma.routine_version_input,
    format: inputItemFormatter(),
    type: 'InputItem' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================