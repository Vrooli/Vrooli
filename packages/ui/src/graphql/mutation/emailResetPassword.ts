import { gql } from 'graphql-tag';
import { sessionFields } from 'graphql/fragment';

export const emailResetPasswordMutation = gql`
    ${sessionFields}
    mutation emailResetPassword($input: EmailResetPasswordInput!) {
    emailResetPassword(input: $input) {
        ...sessionFields
    }
}
`