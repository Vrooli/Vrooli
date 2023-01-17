import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsOrganization' as const;
const suppFields = [] as const;
export const StatsOrganizationModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_organization,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})