import { gql } from 'graphql-tag';
import { notificationSettingsFields } from 'graphql/fragment';

export const notificationSettingsUpdateMutation = gql`
    ${notificationSettingsFields}
    mutation notificationSettingsUpdate($input: NotificationSettingsUpdateInput!) {
        notificationSettingsUpdate(input: $input) {
            ...notificationSettingsFields
        }
    }
`