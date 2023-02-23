import { Prisma } from "@prisma/client";
import { StatsProject, StatsProjectSearchInput, StatsProjectSortBy } from "@shared/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { ProjectModel } from "./project";
import { ModelLogic } from "./types";

const __typename = 'StatsProject' as const;
const suppFields = [] as const;
export const StatsProjectModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsProject,
    GqlSearch: StatsProjectSearchInput,
    GqlSort: StatsProjectSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_projectUpsertArgs['create'],
    PrismaUpdate: Prisma.stats_projectUpsertArgs['update'],
    PrismaModel: Prisma.stats_projectGetPayload<SelectWrap<Prisma.stats_projectSelect>>,
    PrismaSelect: Prisma.stats_projectSelect,
    PrismaWhere: Prisma.stats_projectWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_project,
    display: {
        select: () => ({ id: true, project: selPad(ProjectModel.display.select) }),
        label: (select, languages) => i18next.t(`common:ObjectStats`, {
            lng: languages.length > 0 ? languages[0] : 'en',
            objectName: ProjectModel.display.label(select.project as any, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            project: 'Api',
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsProjectSortBy.DateUpdatedDesc,
        sortBy: StatsProjectSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ project: ProjectModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            project: 'Project',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ProjectModel.validate!.owner(data.project as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_projectSelect>(data, [
            ['project', 'Project'],
        ], languages),
        visibility: {
            private: { project: ProjectModel.validate!.visibility.private },
            public: { project: ProjectModel.validate!.visibility.public },
            owner: (userId) => ({ project: ProjectModel.validate!.visibility.owner(userId) }),
        }
    },
})