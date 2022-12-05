import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const SmartContractModel = ({
    delegate: (prisma: PrismaType) => prisma.smart_contract,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'SmartContract' as GraphQLModelType,
    validate: {} as any,
})