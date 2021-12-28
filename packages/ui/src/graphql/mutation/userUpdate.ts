import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const userUpdateMutation = gql`
    ${sessionFields}
    mutation userUpdate($input: UserUpdateInput!) {
        userUpdate(input: $input) {
            ...sessionFields
        }
    }
`