import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ProjectVersionDirectory, ProjectVersionDirectoryCreateInput, ProjectVersionDirectoryUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { bestLabel } from "../utils";
import { ModelLogic } from "./types";

const __typename = 'ProjectVersionDirectory' as const;
const suppFields = [] as const;
export const ProjectVersionDirectoryModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ProjectVersionDirectoryCreateInput,
    GqlUpdate: ProjectVersionDirectoryUpdateInput,
    GqlModel: ProjectVersionDirectory,
    GqlPermission: {},
    GqlSearch: undefined,
    GqlSort: undefined,
    PrismaCreate: Prisma.project_version_directoryUpsertArgs['create'],
    PrismaUpdate: Prisma.project_version_directoryUpsertArgs['update'],
    PrismaModel: Prisma.project_version_directoryGetPayload<SelectWrap<Prisma.project_version_directorySelect>>,
    PrismaSelect: Prisma.project_version_directorySelect,
    PrismaWhere: Prisma.project_version_directoryWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.project_version_directory,
    display: {
        select: () => ({ id: true, translations: { select: { language: true, name: true } } }),
        label: (select, languages) => bestLabel(select.translations, 'name', languages),
    },
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})