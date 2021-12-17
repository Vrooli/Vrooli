import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const resetPasswordMutation = gql`
    ${userSessionFields}
    mutation resetPassword($input: ResetPasswordInput!) {
    resetPassword(input: $input) {
        ...userSessionFields
    }
}
`