import { gql } from 'graphql-tag';

export const translateQuery = gql`
    query translate($input: TranslateInput!) {
        translate(input: $input) {
            fields
            language
        }
    }
`