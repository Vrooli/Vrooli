import { InputItem } from "../../schema/types";
import { PrismaType } from "types";
import { FormatConverter, GraphQLModelType } from "./base";

//==============================================================
/* #region Custom Components */
//==============================================================

export const inputItemFormatter = (): FormatConverter<InputItem> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.InputItem,
        'standard': GraphQLModelType.Standard,
    },
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function InputItemModel(prisma: PrismaType) {
    const prismaObject = prisma.routine_input;
    const format = inputItemFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
    }
}
//==============================================================
/* #endregion Model */
//==============================================================