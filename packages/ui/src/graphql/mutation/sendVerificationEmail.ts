import { gql } from 'graphql-tag';

export const sendVerificationEmailMutation = gql`
    mutation sendVerificationEmail($input: SendVerificationEmailInput!) {
        sendVerificationEmail(input: $input) {
            success
        }
    }
`