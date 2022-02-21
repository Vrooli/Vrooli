import { gql } from 'graphql-tag';

export const autocompleteQuery = gql`
    fragment autocompleteTagFields on Tag {
        id
        description
        tag
    }
    query autocomplete($input: AutocompleteInput!) {
        autocomplete(input: $input) {
            organizations {
                id
                name
                stars
                isStarred
                tags {
                    ...autocompleteTagFields
                }
            }
            projects {
                id
                name
                stars
                isUpvoted
                isStarred
                tags {
                    ...autocompleteTagFields
                }
            }
            routines {
                id
                title
                stars
                isUpvoted
                isStarred
                tags {
                    ...autocompleteTagFields
                }
            }
            standards {
                id
                name
                stars
                isUpvoted
                isStarred
                tags {
                    ...autocompleteTagFields
                }
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