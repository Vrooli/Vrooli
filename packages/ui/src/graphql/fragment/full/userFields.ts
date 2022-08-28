import { gql } from 'graphql-tag';

export const userFields = gql`
    fragment userFields on User {
        id
        handle
        name
        created_at
        stars
        isStarred
        reportsCount
        resourceLists{
            id
            created_at
            index
            usedFor
            translations {
                id
                language
                description
                title
            }
            resources {
                id
                created_at
                index
                link
                updated_at
                usedFor
                translations {
                    id
                    language
                    description
                    title
                }
            }
        }
        translations {
            id
            language
            bio
        }
    }
`