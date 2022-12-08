import { gql } from 'graphql-tag';
import { noteFields } from 'graphql/fragment';

export const noteQuery = gql`
    ${noteFields}
    query note($input: FindByIdInput!) {
        note(input: $input) {
            ...noteFields
        }
    }
`