import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const emailLogInMutation = gql`
    ${userSessionFields}
    mutation emailLogIn($input: EmailLogInInput) {
    emailLogIn(input: $input) {
        ...userSessionFields
    }
}
`