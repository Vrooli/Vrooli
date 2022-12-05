import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ApiModel = ({
    delegate: (prisma: PrismaType) => prisma.api,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Api' as GraphQLModelType,
    validate: {} as any,
})