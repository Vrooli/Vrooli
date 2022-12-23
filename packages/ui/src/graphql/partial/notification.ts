export const notificationFields = ['Notification', `{
    id
    created_at
    category
    isRead
    title
    description
    link
    imgLink
}`] as const;
export const notificationSettingsFields = ['NotificationSettings', `{
    categories {
        category
        enabled
        dailyLimit
        toEmails
        toSms
        toPush
    }
    dailyLimit
    enabled
    includedEmails
    includedPush
    includedSms
    toEmails
    toPush
    toSms
}`] as const;