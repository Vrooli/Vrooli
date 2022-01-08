import { gql } from 'graphql-tag';

export const autocompleteQuery = gql`
    query autocomplete($input: AutocompleteInput!) {
        autocomplete(input: $input) {
            id
            title
            objectType
            stars
        }
    }
`