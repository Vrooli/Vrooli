import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const RunProjectModel = ({
    delegate: (prisma: PrismaType) => prisma.run_project,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'RunProject' as GraphQLModelType,
    validate: {} as any,
})