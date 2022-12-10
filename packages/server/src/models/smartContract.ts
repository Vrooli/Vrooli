import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { SmartContractVersionModel } from "./smartContractVersion";
import { Displayer } from "./types";

const __typename = 'SmartContract' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.smart_contractSelect,
    Prisma.smart_contractGetPayload<SelectWrap<Prisma.smart_contractSelect>>
> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: SmartContractVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        SmartContractVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const SmartContractModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.smart_contract,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})