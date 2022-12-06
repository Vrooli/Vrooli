import { gql } from 'graphql-tag';

export const notificationMarkAsReadMutation = gql`
    mutation notificationMarkAsRead($input: FindByIdInput!) {
        notificationMarkAsRead(input: $input) {
            success
        }
    }
`