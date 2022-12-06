import { PrismaType } from "../types";
import { GraphQLModelType } from "./types";

export const NotificationModel = ({
    delegate: (prisma: PrismaType) => prisma.notification,
    display: {} as any,
    format: {} as any,
    mutate: {} as any,
    search: {} as any,
    type: 'Notification' as GraphQLModelType,
    validate: {} as any,
})