import { InputItem } from "../schema/types";
import { PrismaType } from "../types";
import { FormatConverter, GraphQLModelType } from "./types";

export const inputItemFormatter = (): FormatConverter<InputItem, any> => ({
    relationshipMap: {
        '__typename': 'InputItem',
        'standard': 'Standard',
    },
})

export const InputItemModel = ({
    prismaObject: (prisma: PrismaType) => prisma.routine_version_input,
    format: inputItemFormatter(),
    type: 'InputItem' as GraphQLModelType,
})