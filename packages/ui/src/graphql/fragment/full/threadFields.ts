import { gql } from 'graphql-tag';

export const threadFields = gql`
    fragment threadComment on Comment {
        id
        created_at
        updated_at
        score
        isUpvoted
        role
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
        translations {
            id
            language
            text
        }
    }
    fragment threadFields on CommentThread {
        childThreads {
            childThreads {
                comment {
                    ...threadComment
                }
                endCursor
                totalInThread
            }
            comment {
                ...threadComment
            }
            endCursor
            totalInThread
        }
        comment {
            ...threadComment
        }
        endCursor
        totalInThread
    }
`