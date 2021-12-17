import { gql } from 'graphql-tag';

export const requestPasswordChangeMutation = gql`
    mutation requestPasswordChange($input: RequestPasswordChangeInput!) {
    requestPasswordChange(input: $input)
}
`