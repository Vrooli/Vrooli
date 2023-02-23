import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Notification, NotificationSearchInput, NotificationSortBy } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";

const __typename = 'Notification' as const;
const suppFields = [] as const;
export const NotificationModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Notification,
    GqlSearch: NotificationSearchInput,
    GqlSort: NotificationSortBy,
    GqlPermission: {},
    PrismaCreate: Prisma.notificationUpsertArgs['create'],
    PrismaUpdate: Prisma.notificationUpsertArgs['update'],
    PrismaModel: Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>,
    PrismaSelect: Prisma.notificationSelect,
    PrismaWhere: Prisma.notificationWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.notification,
    display: {
        select: () => ({ id: true, title: true }),
        label: (select) => select.title,
    },
    format: {
        gqlRelMap: {
            __typename,
        },
        prismaRelMap: {
            __typename,
        },
        countFields: {},
    },
    search: {
        defaultSort: NotificationSortBy.DateCreatedDesc,
        sortBy: NotificationSortBy,
        searchFields: {
            createdTimeFrame: true,
            visibility: true,
        },
        searchStringQuery: () => ({
            OR: [
                'descriptionWrapped',
                'linkWrapped',
                'titleWrapped',
            ]
        }),
    },
    validate: {} as any,
})