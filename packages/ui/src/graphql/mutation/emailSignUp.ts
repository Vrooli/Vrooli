import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const emailSignUpMutation = gql`
    ${userSessionFields}
    mutation emailSignUp($input: EmailSignUpInput!) {
    emailSignUp(input: $input) {
        ...userSessionFields
    }
}
`