import { gql } from 'graphql-tag';

export const deleteUserMutation = gql`
    mutation deleteUser(
        $id: ID!
    ) {
    deleteUser(
        id: $id
    )
}
`