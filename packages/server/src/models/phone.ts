import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const PhoneModel = ({
    delegate: (prisma: PrismaType) => prisma.phone,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Phone' as GraphQLModelType,
    validate: {} as any,
})