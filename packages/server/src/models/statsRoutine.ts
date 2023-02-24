import { Prisma } from "@prisma/client";
import { StatsRoutine, StatsRoutineSearchInput, StatsRoutineSortBy } from "@shared/consts";
import i18next from "i18next";
import { selPad } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, oneIsPublic } from "../utils";
import { RoutineModel } from "./routine";
import { ModelLogic } from "./types";

const __typename = 'StatsRoutine' as const;
const suppFields = [] as const;
export const StatsRoutineModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: StatsRoutine,
    GqlSearch: StatsRoutineSearchInput,
    GqlSort: StatsRoutineSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.stats_routineUpsertArgs['create'],
    PrismaUpdate: Prisma.stats_routineUpsertArgs['update'],
    PrismaModel: Prisma.stats_routineGetPayload<SelectWrap<Prisma.stats_routineSelect>>,
    PrismaSelect: Prisma.stats_routineSelect,
    PrismaWhere: Prisma.stats_routineWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.stats_routine,
    display: {
        select: () => ({ id: true, routine: selPad(RoutineModel.display.select) }),
        label: (select, languages) => (i18next as any).t(`common:ObjectStats`, {
            lng: languages.length > 0 ? languages[0] : 'en',
            objectName: RoutineModel.display.label(select.routine as any, languages),
        }),
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
            routine: 'Routine',
        },
        countFields: {},
    },
    search: {
        defaultSort: StatsRoutineSortBy.DateUpdatedDesc,
        sortBy: StatsRoutineSortBy,
        searchFields: {
            periodTimeFrame: true,
            periodType: true,
        },
        searchStringQuery: () => ({ routine: RoutineModel.search!.searchStringQuery() }),
    },
    validate: {
        isTransferable: false,
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            routine: 'Routine',
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => RoutineModel.validate!.owner(data.routine as any),
        isDeleted: () => false,
        isPublic: (data, languages) => oneIsPublic<Prisma.stats_routineSelect>(data, [
            ['routine', 'Routine'],
        ], languages),
        visibility: {
            private: { routine: RoutineModel.validate!.visibility.private },
            public: { routine: RoutineModel.validate!.visibility.public },
            owner: (userId) => ({ routine: RoutineModel.validate!.visibility.owner(userId) }),
        }
    },
})