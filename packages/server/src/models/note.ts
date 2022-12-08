import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { NoteVersionModel } from "./noteVersion";
import { Displayer, GraphQLModelType } from "./types";

const displayer = (): Displayer<
    Prisma.noteSelect,
    Prisma.noteGetPayload<SelectWrap<Prisma.noteSelect>>
> => ({
    select: () => ({
        id: true,
        versions: {
            orderBy: { versionIndex: 'desc' },
            take: 1,
            select: NoteVersionModel.display.select(),
        }
    }),
    label: (select, languages) => select.versions.length > 0 ?
        NoteVersionModel.display.label(select.versions[0] as any, languages) : '',
})

export const NoteModel = ({
    delegate: (prisma: PrismaType) => prisma.note,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Note' as GraphQLModelType,
    validate: {} as any,
})