import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Notification, NotificationSearchInput, NotificationSortBy } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const type = 'Notification' as const;
const suppFields = [] as const;
export const NotificationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Notification,
    GqlSearch: NotificationSearchInput,
    GqlSort: NotificationSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.notificationUpsertArgs['create'],
    PrismaUpdate: Prisma.notificationUpsertArgs['update'],
    PrismaModel: Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>,
    PrismaSelect: Prisma.notificationSelect,
    PrismaWhere: Prisma.notificationWhereInput,
}, typeof suppFields> = ({
    type,
    delegate: (prisma: PrismaType) => prisma.notification,
    display: {
        select: () => ({ id: true, title: true }),
        label: (select) => select.title,
    },
    format: {
        gqlRelMap: {
            type,
        },
        prismaRelMap: {
            type,
        },
        countFields: {},
    },
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})