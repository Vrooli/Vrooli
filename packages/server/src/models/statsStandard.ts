import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsStandard' as const;
const suppFields = [] as const;
export const StatsStandardModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_standard,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})