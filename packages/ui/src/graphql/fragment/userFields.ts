import { gql } from 'graphql-tag';

export const userFields = gql`
    fragment userFields on User {
        id
        username
        created_at
        stars
        isStarred
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