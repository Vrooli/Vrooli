import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsApi' as const;
const suppFields = [] as const;
export const StatsApiModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_api,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})