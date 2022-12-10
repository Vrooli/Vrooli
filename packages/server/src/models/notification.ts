import { Prisma } from "@prisma/client";
import { SelectWrap } from "../builders/types";
import { Notification } from "../endpoints/types";
import { PrismaType } from "../types";
import { Displayer, Formatter } from "./types";

const __typename = 'Notification' as const;

const suppFields = [] as const;
const formatter = (): Formatter<Notification, typeof suppFields> => ({
    relationshipMap: {
        __typename,
    },
})

const displayer = (): Displayer<
    Prisma.notificationSelect,
    Prisma.notificationGetPayload<SelectWrap<Prisma.notificationSelect>>
> => ({
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