import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const profileUpdateMutation = gql`
    ${sessionFields}
    mutation profileUpdate($input: ProfileUpdateInput!) {
        profileUpdate(input: $input) {
            ...sessionFields
        }
    }
`