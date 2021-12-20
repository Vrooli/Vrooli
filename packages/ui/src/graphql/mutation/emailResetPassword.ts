import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const emailResetPasswordMutation = gql`
    ${userSessionFields}
    mutation emailResetPassword($input: EmailResetPasswordInput!) {
    emailResetPassword(input: $input) {
        ...userSessionFields
    }
}
`