import { gql } from 'graphql-tag';
import { customerSessionFields } from 'graphql/fragment';

export const loginMutation = gql`
    ${customerSessionFields}
    mutation login(
        $email: String
        $password: String
    ) {
    login(
        email: $email
        password: $password
    ) {
        ...customerSessionFields
    }
}
`