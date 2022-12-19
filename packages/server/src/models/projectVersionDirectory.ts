import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { Displayer, ModelLogic } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ProjectVersionDirectoryCreateInput,
    GqlUpdate: ProjectVersionDirectoryUpdateInput,
    GqlModel: ProjectVersionDirectory,
    GqlPermission: any,
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.project_version_directoryUpsertArgs['create'],
    PrismaUpdate: Prisma.project_version_directoryUpsertArgs['update'],
    PrismaModel: Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>,
    PrismaSelect: Prisma.project_version_directorySelect,
    PrismaWhere: Prisma.project_version_directoryWhereInput,
}

const __typename = 'ProjectVersionDirectory' as const;

const suppFields = [] as const;

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
    label: (select, languages) => bestLabel(select.translations, 'name', languages),
})

export const ProjectVersionDirectoryModel: ModelLogic<Model, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: displayer(),
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})