import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsSite' as const;
const suppFields = [] as const;
export const StatsSiteModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_site,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})