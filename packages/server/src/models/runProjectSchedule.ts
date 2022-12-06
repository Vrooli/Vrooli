import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const RunProjectScheduleModel = ({
    delegate: (prisma: PrismaType) => prisma.run_project_schedule,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'RunProjectSchedule' as GraphQLModelType,
    validate: {} as any,
})