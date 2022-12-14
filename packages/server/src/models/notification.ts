import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Notification, NotificationSearchInput, NotificationSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

type Model = {
    IsTransferable: false,
    IsVersioned: false,
    GqlModel: Notification,
    GqlSearch: NotificationSearchInput,
    GqlSort: NotificationSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.notificationUpsertArgs['create'],
    PrismaUpdate: Prisma.notificationUpsertArgs['update'],
    PrismaModel: Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>,
    PrismaSelect: Prisma.notificationSelect,
    PrismaWhere: Prisma.notificationWhereInput,
}

const __typename = 'Notification' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Model, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

const displayer = (): Displayer<Model> => ({
    select: () => ({ id: true, title: true }),
    label: (select) => select.title,
})

export const NotificationModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.notification,
    display: displayer(),
    format: formatter(),
    mutate: {} as any,
    search: {} as any,
    validate: {} as any,
})