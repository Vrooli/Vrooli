import gql from 'graphql-tag';
import { Email_full } from '../fragments/Email_full';
import { Phone_full } from '../fragments/Phone_full';
import { PushDevice_full } from '../fragments/PushDevice_full';

export const notificationSettingsUpdate = gql`${Email_full}
${Phone_full}
${PushDevice_full}

mutation notificationSettingsUpdate($input: NotificationSettingsUpdateInput!) {
  notificationSettingsUpdate(input: $input) {
    includedEmails {
        ...Email_full
    }
    includedSms {
        ...Phone_full
    }
    includedPush {
        ...PushDevice_full
    }
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

