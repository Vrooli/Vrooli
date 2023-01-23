import { Notification, NotificationSettings, NotificationSettingsCategory } from "@shared/consts";
import { relPartial } from '../utils';
import { GqlPartial } from "types";

export const notificationPartial: GqlPartial<Notification> = {
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

export const notificationSettingsCategoryPartial: GqlPartial<NotificationSettingsCategory> = {
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

export const notificationSettingsPartial: GqlPartial<NotificationSettings> = {
    __typename: 'NotificationSettings',
    full: {
        includedEmails: true,
        includedSms: true,
        includedPush: true,
        toEmails: true,
        toSms: true,
        toPush: true,
        dailyLimit: true,
        enabled: true,
        categories: () => relPartial(notificationSettingsCategoryPartial, 'full', { omit: 'notificationSettings' }),
    },
}