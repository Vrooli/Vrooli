import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ReminderList, ReminderListCreateInput, ReminderListSearchInput, ReminderListSortBy, ReminderListUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { UserScheduleModel } from "./userSchedule";

const __typename = 'ReminderList' as const;
const suppFields = [] as const;
export const ReminderListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderListCreateInput,
    GqlUpdate: ReminderListUpdateInput,
    GqlModel: ReminderList,
    GqlSearch: ReminderListSearchInput,
    GqlSort: ReminderListSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.reminder_listUpsertArgs['create'],
    PrismaUpdate: Prisma.reminder_listUpsertArgs['update'],
    PrismaModel: Prisma.reminder_listGetPayload<SelectWrap<Prisma.reminder_listSelect>>,
    PrismaSelect: Prisma.reminder_listSelect,
    PrismaWhere: Prisma.reminder_listWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.reminder_list,
    display: {
        select: () => ({ id: true, userSchedule: { select: UserScheduleModel.display.select() } }),
        // Label is schedule's label
        label: (select, languages) =>  UserScheduleModel.display.label(select.userSchedule as any, languages),
    },
    format: {
        gqlRelMap: {
            __typename,
            userSchedule: 'UserSchedule',
            reminders: 'Reminder',
        },
        prismaRelMap: {
            __typename,
            userSchedule: 'UserSchedule',
            reminders: 'Reminder',
        },
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})