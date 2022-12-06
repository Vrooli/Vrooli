import { gql } from 'graphql-tag';

export const notificationsQuery = gql`
    query notifications($input: NotificationSearchInput!) {
        notifications(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    id
                    created_at
                    category
                    isRead
                    title
                    description
                    link
                    imgLink
                }
            }
        }
    }
`