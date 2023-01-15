import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'StatsSmartContract' as const;
const suppFields = [] as const;
export const StatsSmartContractModel: ModelLogic<any, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_smart_contract,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})