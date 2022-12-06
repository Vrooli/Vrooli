import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const NotificationSubscriptionModel = ({
    delegate: (prisma: PrismaType) => prisma.notification_subscription,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'NotificationSubscription' as GraphQLModelType,
    validate: {} as any,
})