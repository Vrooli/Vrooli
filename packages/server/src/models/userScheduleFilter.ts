import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { MaxObjects, UserScheduleFilter, UserScheduleFilterCreateInput, UserScheduleSearchInput, UserScheduleSortBy } from '@shared/consts';
import { PrismaType } from "../types";
import { TagModel } from "./tag";
import { ModelLogic } from "./types";
import { UserScheduleModel } from "./userSchedule";
import { defaultPermissions } from "../utils";

const __typename = 'UserScheduleFilter' as const;
const suppFields = [] as const;
export const UserScheduleFilterModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: UserScheduleFilterCreateInput,
    GqlUpdate: undefined,
    GqlModel: UserScheduleFilter,
    GqlPermission: {},
    GqlSearch: UserScheduleSearchInput,
    GqlSort: UserScheduleSortBy,
    PrismaCreate: Prisma.user_schedule_filterUpsertArgs['create'],
    PrismaUpdate: Prisma.user_schedule_filterUpsertArgs['update'],
    PrismaModel: Prisma.user_schedule_filterGetPayload<SelectWrap<Prisma.user_schedule_filterSelect>>,
    PrismaSelect: Prisma.user_schedule_filterSelect,
    PrismaWhere: Prisma.user_schedule_filterWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user_schedule_filter,
    display: {
        select: () => ({ id: true, tag: { select: TagModel.display.select() } }),
        label: (select, languages) => select.tag ? TagModel.display.label(select.tag as any, languages) : '',
    },
    format: {
        gqlRelMap: {
            __typename,
            userSchedule: 'UserSchedule',
            tag: 'Tag',
        },
        prismaRelMap: {
            __typename,
            userSchedule: 'UserSchedule',
            tag: 'Tag',
        },
        countFields: {},
    },
    mutate: {} as any,
    validate: {
        isDeleted: () => false,
        isPublic: () => false,
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => UserScheduleModel.validate!.owner(data.userSchedule as any),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            userSchedule: 'UserSchedule',
        }),
        visibility: {
            private: {},
            public: {},
            owner: (userId) => ({
                userSchedule: UserScheduleModel.validate!.visibility.owner(userId),
            }),
        },
    },
})