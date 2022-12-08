import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { ApiVersionModel } from "./apiVersion";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.apiSelect,
    Prisma.apiGetPayload<SelectWrap<Prisma.apiSelect>>
> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: ApiVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        ApiVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const ApiModel = ({
    delegate: (prisma: PrismaType) => prisma.api,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Api' as GraphQLModelType,
    validate: {} as any,
})