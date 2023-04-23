import { logger } from "../events";
const defaultSettings = { __typename: "NotificationSettings", enabled: false };
export const parseNotificationSettings = (settingsJson) => {
    try {
        const settings = settingsJson ? JSON.parse(settingsJson) : defaultSettings;
        return settings;
    }
    catch (error) {
        logger.error("Failed to parse notification settings", { trace: "0304" });
        return { __typename: "NotificationSettings", enabled: false };
    }
};
export const getNotificationSettingsAndRecipients = async (prisma, userIds) => {
    const results = [];
    const usersData = await prisma.user.findMany({
        where: { id: { in: userIds } },
        select: {
            id: true,
            notificationSettings: true,
            pushDevices: true,
            emails: true,
        },
    });
    if (!usersData)
        return Array(userIds.length).fill({ settings: defaultSettings, pushDevices: [], emails: [], phoneNumbers: [] });
    for (let i = 0; i < userIds.length; i++) {
        const user = usersData.find(u => u.id === userIds[i]);
        if (!user) {
            results.push({ settings: defaultSettings, pushDevices: [], emails: [], phoneNumbers: [] });
            continue;
        }
        const settings = parseNotificationSettings(user.notificationSettings);
        results.push({
            settings,
            pushDevices: user.pushDevices,
            emails: user.emails,
            phoneNumbers: [],
        });
    }
    return results;
};
export const updateNotificationSettings = async (settings, prisma, userId) => {
    const settingsAndData = await getNotificationSettingsAndRecipients(prisma, [userId]);
    const currentSettings = settingsAndData[0].settings;
    const newSettings = { ...currentSettings, ...settings };
    if (settings.categories) {
        newSettings.categories = { ...currentSettings.categories, ...settings.categories };
        for (const category in settings.categories) {
            newSettings.categories[category] = { ...(currentSettings.categories?.[category] ?? {}), ...settings.categories[category] };
        }
    }
    const updated = await prisma.user.update({
        where: { id: userId },
        data: { notificationSettings: JSON.stringify(newSettings) },
        select: { notificationSettings: true },
    });
    return parseNotificationSettings(updated.notificationSettings);
};
export const findRecipientsAndLimit = async (category, prisma, userIds) => {
    const result = [];
    const allData = await getNotificationSettingsAndRecipients(prisma, userIds);
    for (let i = 0; i < userIds.length; i++) {
        const userResult = { pushDevices: [], emails: [], phoneNumbers: [] };
        const { settings, pushDevices, emails, phoneNumbers } = allData[i];
        if (!settings.enabled) {
            result.push(userResult);
            continue;
        }
        if (settings.dailyLimit)
            userResult.dailyLimit = settings.dailyLimit;
        if (settings.categories && settings.categories[category] && !settings.categories[category].enabled) {
            result.push(userResult);
            continue;
        }
        if (settings.includedPush && settings.toPush !== false) {
            userResult.pushDevices = pushDevices.filter(device => settings.includedPush?.some(({ id }) => id === device.id));
        }
        else {
            userResult.pushDevices = pushDevices;
        }
        if (settings.includedEmails && settings.toEmails !== false) {
            userResult.emails = emails.filter(email => settings.includedEmails?.some(({ id }) => id === email.id));
        }
        else {
            userResult.emails = emails;
        }
        if (settings.includedSms && settings.toSms !== false) {
            userResult.phoneNumbers = phoneNumbers.filter(phoneNumber => settings.includedSms?.includes(phoneNumber.id));
        }
        else {
            userResult.phoneNumbers = phoneNumbers;
        }
        if (settings.categories && settings.categories[category]) {
            const categoryData = settings.categories[category];
            if (categoryData.toPush === false)
                userResult.pushDevices = [];
            if (categoryData.toEmails === false)
                userResult.emails = [];
            if (categoryData.toSms === false)
                userResult.phoneNumbers = [];
            if (categoryData.dailyLimit)
                userResult.dailyLimit = categoryData.dailyLimit;
        }
        result.push(userResult);
    }
    return result;
};
//# sourceMappingURL=notificationSettings.js.map