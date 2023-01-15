import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsApi' as const;
const suppFields = [] as const;
export const StatsApiModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_api,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})