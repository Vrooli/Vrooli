import { gql } from 'graphql-tag';

export const requestPasswordChangeMutation = gql`
    mutation requestPasswordChange(
        $email: String!
    ) {
    requestPasswordChange(
        email: $email
    )
}
`