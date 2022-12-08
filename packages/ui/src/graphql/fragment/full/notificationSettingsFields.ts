import { gql } from 'graphql-tag';

export const notificationSettingsFields = gql`
    fragment notificationSettingsFields on NotificationSettings {
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
    }
`