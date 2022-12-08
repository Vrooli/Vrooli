import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.smart_contract_versionSelect,
    Prisma.smart_contract_versionGetPayload<SelectWrap<Prisma.smart_contract_versionSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const SmartContractVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.smart_contract_version,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'SmartContractVersion' as GraphQLModelType,
    validate: {} as any,
})