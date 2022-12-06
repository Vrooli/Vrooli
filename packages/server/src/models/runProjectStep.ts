import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const RunProjectStepModel = ({
    delegate: (prisma: PrismaType) => prisma.run_project_step,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'RunProjectStep' as GraphQLModelType,
    validate: {} as any,
})