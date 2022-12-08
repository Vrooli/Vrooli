import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.api_versionSelect,
    Prisma.api_versionGetPayload<SelectWrap<Prisma.api_versionSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => {
        // Return name if exists, or callLink host
        const name = bestLabel(select.translations, 'name', languages)
        if (name.length > 0) return name
        const url = new URL(select.callLink)
        return url.host
    },
})

export const ApiVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.api_version,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'ApiVersion' as GraphQLModelType,
    validate: {} as any,
})