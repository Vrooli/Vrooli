import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const updateUserMutation = gql`
    ${sessionFields}
    mutation updateUser($input: UpdateUserInput!) {
    updateUser(input: $input) {
        ...sessionFields
    }
}
`