import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const RunRoutineScheduleModel = ({
    delegate: (prisma: PrismaType) => prisma.run_routine_schedule,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'RunRoutineSchedule' as GraphQLModelType,
    validate: {} as any,
})