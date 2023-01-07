import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsRoutine' as const;
const suppFields = [] as const;
export const StatsRoutineModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_routine,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})