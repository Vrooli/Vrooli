import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.note_versionSelect,
    Prisma.note_versionGetPayload<SelectWrap<Prisma.note_versionSelect>>
> => ({
    select: () => ({ id: true, callLink: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages)
})

export const NoteVersionModel = ({
    delegate: (prisma: PrismaType) => prisma.note_version,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'NoteVersion' as GraphQLModelType,
    validate: {} as any,
})