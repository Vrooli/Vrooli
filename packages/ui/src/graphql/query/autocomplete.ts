import { gql } from 'graphql-tag';

export const autocompleteQuery = gql`
    query autocomplete($input: AutocompleteInput!) {
        autocomplete(input: $input) {
            organizations {
                id
                name
                stars
                isStarred
            }
            projects {
                id
                name
                stars
                isUpvoted
                isStarred
            }
            routines {
                id
                title
                stars
                isUpvoted
                isStarred
            }
            standards {
                id
                name
                stars
                isUpvoted
                isStarred
            }
            users {
                id
                username
                stars
                isStarred
            }
        }
    }
`