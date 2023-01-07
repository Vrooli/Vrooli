import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsStandard' as const;
const suppFields = [] as const;
export const StatsStandardModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_standard,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})