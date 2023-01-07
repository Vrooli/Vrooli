import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsSite' as const;
const suppFields = [] as const;
export const StatsSiteModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_site,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})