import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const LabelModel = ({
    delegate: (prisma: PrismaType) => prisma.label,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Label' as GraphQLModelType,
    validate: {} as any,
})