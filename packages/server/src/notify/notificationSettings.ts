import { Prisma } from "@prisma/client";
import { NotificationSettings, NotificationSettingsUpdateInput } from '@shared/consts';
import { logger } from "../events";
import { PrismaType } from "../types";
import { NotificationCategory } from "./notify";

type NotificationRecipients = {
    pushDevices: Prisma.push_deviceGetPayload<{ select: { [K in keyof Required<Omit<Prisma.push_deviceSelect, 'user'>>]: true } }>[],
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
 * Finds the current notification settings for a list of users
 * @param prisma Prisma client
 * @param userIds IDs of the users
 * @returns Notification settings for each user, in the same order as the IDs
 */
export const getNotificationSettingsAndRecipients = async (prisma: PrismaType, userIds: string[]): Promise<Array<{ settings: NotificationSettings } & NotificationRecipients>> => {
    // Initialize results array
    const results: Array<{ settings: NotificationSettings } & NotificationRecipients> = [];
    // Get the current notification settings for each user
    const usersData = await prisma.user.findMany({
        where: { id: { in: userIds } },
        select: {
            id: true,
            notificationSettings: true,
            pushDevices: true,
            emails: true,
        }
    });
    // If no results, return default settings
    if (!usersData) return Array(userIds.length).fill({ settings: defaultSettings, pushDevices: [], emails: [], phoneNumbers: [] });
    // For each user, parse the notification settings and add to the results array
    for (let i = 0; i < userIds.length; i++) {
        // Find queried user data
        const user = usersData.find(u => u.id === userIds[i]);
        // If no user data, return default settings
        if (!user) {
            results.push({ settings: defaultSettings, pushDevices: [], emails: [], phoneNumbers: [] });
            continue;
        }
        // Parse the notification settings
        const settings = parseNotificationSettings(user.notificationSettings);
        // Add to results array
        results.push({
            settings,
            pushDevices: user.pushDevices,
            emails: user.emails,
            phoneNumbers: [],
        });
    }
    return results;
}

/**
 * Updates the notification settings of one user
 * @param settings The new notification settings, or partial notification settings
 * @param prisma The prisma client
 * @param userId The id of the user
 * @returns The updated notification settings
 */
export const updateNotificationSettings = async (
    settings: NotificationSettingsUpdateInput,
    prisma: PrismaType,
    userId: string): Promise<NotificationSettings> => {
    // Get the current notification settings
    const settingsAndData = await getNotificationSettingsAndRecipients(prisma, [userId]);
    const currentSettings = settingsAndData[0].settings;
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
    const updated = await prisma.user.update({
        where: { id: userId },
        data: { notificationSettings: JSON.stringify(newSettings) },
        select: { notificationSettings: true }
    });
    return parseNotificationSettings(updated.notificationSettings);
}

/**
 * Finds out which devices, emails, and numbers to send a notification to. Also 
 * finds out maximum number of notifications that can be sent in a day.
 * @param category The category of the notification
 * @param prisma The prisma client
 * @param userIds The ids of the users
 */
export const findRecipientsAndLimit = async (
    category: NotificationCategory,
    prisma: PrismaType,
    userIds: string[]
): Promise<Array<NotificationRecipients & { dailyLimit?: number }>> => {
    // Initialize the return object
    const result: Array<NotificationRecipients & { dailyLimit?: number }> = [];
    // Get the current notification settings and recipients
    let allData = await getNotificationSettingsAndRecipients(prisma, userIds);
    // For each user, find the recipients and daily limit
    for (let i = 0; i < userIds.length; i++) {
        // Initialize the return object for this user
        const userResult: NotificationRecipients & { dailyLimit?: number } = { pushDevices: [], emails: [], phoneNumbers: [] };
        const { settings, pushDevices, emails, phoneNumbers } = allData[i];
        // If the user has disabled notifications, return empty arrays
        if (!settings.enabled) {
            result.push(userResult);
            continue;
        }
        // Set limit
        if (settings.dailyLimit) userResult.dailyLimit = settings.dailyLimit;
        // If the category is not enabled, return empty arrays
        if (settings.categories && settings.categories[category] && !settings.categories[category].enabled) {
            result.push(userResult);
            continue;
        }
        // Add included devices, emails, and numbers to the return object
        if (settings.includedPush && settings.toPush !== false) {
            userResult.pushDevices = pushDevices.filter(device => settings.includedPush?.includes(device.id));
        } else {
            userResult.pushDevices = pushDevices;
        }
        if (settings.includedEmails && settings.toEmails !== false) {
            userResult.emails = emails.filter(email => settings.includedEmails?.includes(email.id));
        } else {
            userResult.emails = emails;
        }
        if (settings.includedSms && settings.toSms !== false) {
            userResult.phoneNumbers = phoneNumbers.filter(phoneNumber => settings.includedSms?.includes(phoneNumber.id));
        } else {
            userResult.phoneNumbers = phoneNumbers;
        }
        // Exclude any recipient types that are not enabled for the category
        if (settings.categories && settings.categories[category]) {
            const categoryData = settings.categories[category];
            if (categoryData.toPush === false) userResult.pushDevices = [];
            if (categoryData.toEmails === false) userResult.emails = [];
            if (categoryData.toSms === false) userResult.phoneNumbers = [];
            if (categoryData.dailyLimit) userResult.dailyLimit = categoryData.dailyLimit;
        }
        // Add to results array
        result.push(userResult);
    }
    return result;
}