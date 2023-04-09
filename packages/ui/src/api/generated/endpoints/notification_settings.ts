import gql from 'graphql-tag';
import { Email_full } from '../fragments/Email_full';
import { Phone_full } from '../fragments/Phone_full';
import { PushDevice_full } from '../fragments/PushDevice_full';

export const notificationSettings = gql`${Email_full}
${Phone_full}
${PushDevice_full}

query notificationSettings {
  notificationSettings {
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

