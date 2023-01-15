import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsRoutine' as const;
const suppFields = [] as const;
export const StatsRoutineModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_routine,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})