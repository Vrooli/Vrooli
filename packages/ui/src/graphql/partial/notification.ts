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
    includedEmails
    includedSms
    includedPush
    toEmails
    toSms
    toPush
    dailyLimit
    enabled
    categories
}`] as const;