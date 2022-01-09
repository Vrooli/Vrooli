import { gql } from 'graphql-tag';

export const autocompleteQuery = gql`
    query autocomplete($input: AutocompleteInput!) {
        autocomplete(input: $input) {
            organizations {
                id
                name
                stars
            }
            projects {
                id
                name
                stars
            }
            routines {
                id
                title
                stars
            }
            standards {
                id
                name
                stars
            }
            users {
                id
                username
                stars
            }
        }
    }
`