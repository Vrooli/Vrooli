import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const ApiVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.api_version,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ApiVersion' as GraphQLModelType,
    validate: {} as any,
})