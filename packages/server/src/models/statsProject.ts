import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsProject' as const;
const suppFields = [] as const;
export const StatsProjectModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_project,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})