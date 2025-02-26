import { Notification, NotificationSettings, NotificationSettingsCategory } from "@local/shared";
import { ApiPartial } from "../types.js";
import { rel } from "../utils.js";

export const notification: ApiPartial<Notification> = {
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
};

export const notificationSettingsCategory: ApiPartial<NotificationSettingsCategory> = {
    full: {
        category: true,
        enabled: true,
        dailyLimit: true,
        toEmails: true,
        toSms: true,
        toPush: true,
    },
};

export const notificationSettings: ApiPartial<NotificationSettings> = {
    full: {
        includedEmails: async () => rel((await import("./email.js")).email, "list"),
        includedSms: async () => rel((await import("./phone.js")).phone, "list"),
        includedPush: async () => rel((await import("./pushDevice.js")).pushDevice, "list"),
        toEmails: true,
        toSms: true,
        toPush: true,
        dailyLimit: true,
        enabled: true,
        categories: () => rel(notificationSettingsCategory, "full"),
    },
};
