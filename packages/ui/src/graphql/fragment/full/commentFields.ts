import { gql } from 'graphql-tag';

export const commentFields = gql`
    fragment commentFields on Comment {
        id
        created_at
        updated_at
        score
        isUpvoted
        isStarred
        commentedOn {
            ... on Project {
                id
                handle
                translations {
                    id
                    language
                    name
                }
            }
            ... on Routine {
                id
                complexity
                translations {
                    id
                    language
                    title
                }
            }
            ... on Standard {
                id
                name
                type
            }
        }
        creator {
            ... on Organization {
                id
                handle
                translations {
                    id
                    language
                    name
                }
            }
            ... on User {
                id
                name
                handle
            }
        }
        permissionsComment {
            canDelete
            canEdit
            canStar
            canReply
            canReport
            canVote
        }
        translations {
            id
            language
            text
        }
    }
`