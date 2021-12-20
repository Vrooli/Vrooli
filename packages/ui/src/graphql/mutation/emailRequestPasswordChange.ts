import { gql } from 'graphql-tag';

export const emailRequestPasswordChangeMutation = gql`
    mutation emailRequestPasswordChange($input: EmailRequestPasswordChangeInput!) {
    emailRequestPasswordChange(input: $input)
}
`