import gql from 'graphql-tag';

export const notificationSettingsUpdate = gql`
mutation notificationSettingsUpdate($input: NotificationSettingsUpdateInput!) {
  notificationSettingsUpdate(input: $input) {
    includedEmails
    includedSms
    includedPush
    toEmails
    toSms
    toPush
    dailyLimit
    enabled
    categories {
        category
        enabled
        dailyLimit
        toEmails
        toSms
        toPush
    }
  }
}`;

