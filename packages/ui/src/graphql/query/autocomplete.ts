import { gql } from 'graphql-tag';

export const autocompleteQuery = gql`
    fragment autocompleteTagFields on Tag {
        id
        created_at
        isStarred
        stars
        tag
        translations {
            id
            language
            description
        }
    }
    query autocomplete($input: AutocompleteInput!) {
        autocomplete(input: $input) {
            organizations {
                id
                stars
                isStarred
                tags {
                    ...autocompleteTagFields
                }
                translations { 
                    id
                    language
                    name
                    bio
                }
            }
            projects {
                id
                score
                stars
                isUpvoted
                isStarred
                tags {
                    ...autocompleteTagFields
                }
                translations {
                    id
                    language
                    name
                    description
                }
            }
            routines {
                id
                score
                stars
                isUpvoted
                isStarred
                tags {
                    ...autocompleteTagFields
                }
                translations {
                    id
                    language
                    title
                    description
                    instructions
                }
            }
            standards {
                id
                score
                stars
                isUpvoted
                isStarred
                name
                tags {
                    ...autocompleteTagFields
                }
                translations {
                    id
                    language
                    description
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