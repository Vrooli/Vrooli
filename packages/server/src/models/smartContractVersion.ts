import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const SmartContractVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.smart_contract_version,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'SmartContractVersion' as GraphQLModelType,
    validate: {} as any,
})