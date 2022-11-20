import { Prisma } from "@prisma/client";
import { logger } from "../events";
import { PrismaType, RecursivePartial } from "../types";
import { NotificationCategory } from "./notify";

type NotificationSettingsField = {
    enabled: boolean;
    dailyLimit: number;
    toEmails?: boolean;
    toSms?: boolean;
    toPush?: boolean;
}

export type NotificationSettings = {
    includedEmails?: string[];
    includedSms?: string[];
    includedDeviceIds?: string[];
    // Add disabled indiicators for each recipient type, 
    // to allow us to disable and reenable notifications
    // without forgetting the previous recipients
    toEmails?: boolean;
    toSms?: boolean;
    toPush?: boolean;
    dailyLimit?: number;
    enabled: boolean;
    categories?: {
        [key in NotificationCategory]: NotificationSettingsField;
    }
}

type NotificationRecipients = {
    pushDevices: Prisma.notification_deviceGetPayload<{ select: { [K in keyof Required<Omit<Prisma.notification_deviceSelect, 'user'>>]: true } }>[],
    emails: Prisma.emailGetPayload<{ select: { [K in keyof Required<Omit<Prisma.emailSelect, 'user'>>]: true } }>[],
    phoneNumbers: any,
}

const defaultSettings = { enabled: false };

/**
 * Parses a user's notification settings from the a stringified JSON object
 * @param settingsJson JSON string of the user's notification settings
 * @returns Parsed notification settings
 */
export const parseNotificationSettings = (settingsJson: string | null): NotificationSettings => {
    try {
        const settings = settingsJson ? JSON.parse(settingsJson) : defaultSettings;
        return settings;
    } catch (error) {
        logger.error(`Failed to parse notification settings`, { trace: '0304' });
        // If there is an error parsing the JSON, return the default settings
        return { enabled: false }
    }
}

/**
 * Finds the current notification settings for a user
 * @param prisma Prisma client
 * @param userId ID of the user
 * @returns Notification settings for the user
 */
export const getNotificationSettingsAndRecipients = async (prisma: PrismaType, userId: string): Promise<{
    settings: NotificationSettings,
} & NotificationRecipients> => {
    // Get the current notification settings
    const user = await prisma.user.findUnique({
        where: { id: userId },
        select: { 
            notificationSettings: true,
            notificationDevices: true,
            emails: true,
        }
    });
    const settings: NotificationSettings = user?.notificationSettings ? parseNotificationSettings(user.notificationSettings) : defaultSettings;
    return {
        settings,
        pushDevices: user?.notificationDevices ?? [],
        emails: user?.emails ?? [],
        phoneNumbers: [],
    }
}

/**
 * Updates the notification settings of a user
 * @param settings The new notification settings, or partial notification settings
 * @param prisma The prisma client
 * @param userId The id of the user
 * @returns The updated notification settings
 */
export const updateNotificationSettings = async (
    settings: RecursivePartial<NotificationSettings>,
    prisma: PrismaType,
    userId: string): Promise<NotificationSettings> => {
    // Get the current notification settings
    const { settings: currentSettings } = await getNotificationSettingsAndRecipients(prisma, userId);
    // Merge the new settings with the current settings, making sure to handle nested objects.
    // For example, { category: { 'Transfer': { enabled: true } } } merged with 
    // { category: { 'Transfer': { dailyLimit: 10 } } } should result in
    // { category: { 'Transfer': { enabled: true, dailyLimit: 10 } } }
    const newSettings = { ...currentSettings, ...settings };
    if (settings.categories) {
        newSettings.categories = { ...currentSettings.categories, ...settings.categories };
        for (const category in settings.categories) {
            newSettings.categories[category] = { ...(currentSettings.categories?.[category] ?? {}), ...settings.categories[category] };
        }
    }
    // Update the user's notification settings
    await prisma.user.update({
        where: { id: userId },
        data: { notificationSettings: JSON.stringify(newSettings) }
    });
    return newSettings;

}

/**
 * Finds out which devices, emails, and numbers to send a notification to. Also 
 * finds out maximum number of notifications that can be sent in a day.
 * @param category The category of the notification
 * @param prisma The prisma client
 * @param userId The id of the user
 */
export const findRecipientsAndLimit = async (
    category: NotificationCategory,
    prisma: PrismaType,
    userId: string
): Promise<NotificationRecipients & { dailyLimit?: number }> => {
    // Initialize the return object
    const result: NotificationRecipients & { dailyLimit?: number } = { pushDevices: [], emails: [], phoneNumbers: [] };
    // Get the current notification settings and recipients
    let { settings, pushDevices, emails, phoneNumbers } = await getNotificationSettingsAndRecipients(prisma, userId);
    // If the user has disabled notifications, return empty arrays
    if (!settings.enabled) return result;
    // Set limit
    if (settings.dailyLimit) result.dailyLimit = settings.dailyLimit;
    // If the category is not enabled, return empty arrays
    if (settings.categories && settings.categories[category] && !settings.categories[category].enabled) return result;
    // Add included devices, emails, and numbers to the return object
    if (settings.includedDeviceIds && settings.toPush !== false) {
        result.pushDevices = pushDevices.filter(device => settings.includedDeviceIds?.includes(device.id));
    } else {
        result.pushDevices = pushDevices;
    }
    if (settings.includedEmails && settings.toEmails !== false) {
        result.emails = emails.filter(email => settings.includedEmails?.includes(email.id));
    } else {
        result.emails = emails;
    }
    if (settings.includedSms && settings.toSms !== false) {
        result.phoneNumbers = phoneNumbers.filter(phoneNumber => settings.includedSms?.includes(phoneNumber.id));
    } else {
        result.phoneNumbers = phoneNumbers;
    }
    // Exclude any recipient types that are not enabled for the category
    if (settings.categories && settings.categories[category]) {
        const categoryData = settings.categories[category];
        if (categoryData.toPush === false) result.pushDevices = [];
        if (categoryData.toEmails === false) result.emails = [];
        if (categoryData.toSms === false) result.phoneNumbers = [];
        if (categoryData.dailyLimit) result.dailyLimit = categoryData.dailyLimit;
    }
    return result;
}