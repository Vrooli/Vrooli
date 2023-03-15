import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { ReminderList, ReminderListCreateInput, ReminderListUpdateInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { UserScheduleModel } from "./userSchedule";
import { reminderListValidation } from "@shared/validation";
import { shapeHelper } from "../builders";

const __typename = 'ReminderList' as const;
const suppFields = [] as const;
export const ReminderListModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: ReminderListCreateInput,
    GqlUpdate: ReminderListUpdateInput,
    GqlModel: ReminderList,
    GqlSearch: undefined,
    GqlSort: undefined,
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
    mutate: {
        shape: {
            create: async ({ data, ...rest }) => ({
                id: data.id,
                ...(await shapeHelper({ relation: 'userSchedule', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'UserSchedule', parentRelationshipName: 'reminderList', data, ...rest })),
                ...(await shapeHelper({ relation: 'reminders', relTypes: ['Create'], isOneToOne: false, isRequired: false, objectType: 'Reminder', parentRelationshipName: 'reminderList', data, ...rest })),
            }),
            update: async ({ data, ...rest }) => ({
                ...(await shapeHelper({ relation: 'userSchedule', relTypes: ['Connect'], isOneToOne: true, isRequired: false, objectType: 'UserSchedule', parentRelationshipName: 'reminderList', data, ...rest })),
                ...(await shapeHelper({ relation: 'reminders', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'Reminder', parentRelationshipName: 'reminderList', data, ...rest })),
            }),
        },
        yup: reminderListValidation,
    },
    validate: {} as any,
})