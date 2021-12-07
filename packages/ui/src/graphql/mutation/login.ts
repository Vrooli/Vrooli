import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const loginMutation = gql`
    ${userSessionFields}
    mutation login(
        $email: String
        $password: String
    ) {
    login(
        email: $email
        password: $password
    ) {
        ...userSessionFields
    }
}
`