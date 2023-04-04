import { Notification, NotificationSettings, NotificationSettingsCategory } from "@shared/consts";
import { GqlPartial } from "../types";
import { rel } from '../utils';

export const notification: GqlPartial<Notification> = {
    __typename: 'Notification',
    full: {
        id: true,
        created_at: true,
        category: true,
        isRead: true,
        title: true,
        description: true,
        link: true,
        imgLink: true,
    },
}

export const notificationSettingsCategory: GqlPartial<NotificationSettingsCategory> = {
    __typename: 'NotificationSettingsCategory',
    full: {
        category: true,
        enabled: true,
        dailyLimit: true,
        toEmails: true,
        toSms: true,
        toPush: true,
    },
}

export const notificationSettings: GqlPartial<NotificationSettings> = {
    __typename: 'NotificationSettings',
    full: {
        __define: {
            0: async () => rel((await import('./email')).email, 'list'),
            1: async () => rel((await import('./phone')).phone, 'list'),
            2: async () => rel((await import('./pushDevice')).pushDevice, 'list'),
        },
        includedEmails: { __use: 0 },
        includedSms: { __use: 1 },
        includedPush: { __use: 2 },
        toEmails: true,
        toSms: true,
        toPush: true,
        dailyLimit: true,
        enabled: true,
        categories: () => rel(notificationSettingsCategory, 'full'),
    },
}