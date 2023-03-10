import gql from 'graphql-tag';

export const notificationSettings = gql`
query notificationSettings {
  notificationSettings {
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

