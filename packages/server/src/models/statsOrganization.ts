import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'StatsOrganization' as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<any, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.stats_organization,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})