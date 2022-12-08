import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.issueSelect,
    Prisma.issueGetPayload<SelectWrap<Prisma.issueSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const IssueModel = ({
    delegate: (prisma: PrismaType) => prisma.issue,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Issue' as GraphQLModelType,
    validate: {} as any,
})