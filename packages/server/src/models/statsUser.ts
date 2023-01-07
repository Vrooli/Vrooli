import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsUser' as const;
const suppFields = [] as const;
export const StatsUserModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_user,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})