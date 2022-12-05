import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ApiKeyModel = ({
    delegate: (prisma: PrismaType) => prisma.api_key,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ApiKey' as GraphQLModelType,
    validate: {} as any,
})