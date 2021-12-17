import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const logInMutation = gql`
    ${userSessionFields}
    mutation logIn(
        $email: String
        $password: String
    ) {
    logIn(
        email: $email
        password: $password
    ) {
        ...userSessionFields
    }
}
`