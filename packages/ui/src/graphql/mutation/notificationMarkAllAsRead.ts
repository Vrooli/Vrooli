import { gql } from 'graphql-tag';

export const notificationMarkAllAsReadMutation = gql`
    mutation notificationMarkAllAsRead {
        notificationMarkAllAsRead {
            count
        }
    }
`