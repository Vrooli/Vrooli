import { gql } from 'graphql-tag';

export const deleteUserMutation = gql`
    mutation deleteUser($input: DeleteUserInput!) {
    deleteUser(input: $input)
}
`