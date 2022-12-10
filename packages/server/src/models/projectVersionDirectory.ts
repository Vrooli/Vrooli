import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer } from "./types";

const __typename = 'ProjectVersionDirectory' as const;

const suppFields = [] as const;

const displayer = (): Displayer<
    Prisma.project_version_directorySelect,
    Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>
> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const ProjectVersionDirectoryModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})