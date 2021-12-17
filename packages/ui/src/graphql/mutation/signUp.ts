import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const signUpMutation = gql`
    ${userSessionFields}
    mutation signUp($input: SignUpInput!) {
    signUp(input: $input) {
        ...userSessionFields
    }
}
`