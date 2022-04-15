import { gql } from 'graphql-tag';

export const walletDeleteOneMutation = gql`
    mutation walletDeleteOne($input: DeleteOneInput!) {
        walletDeleteOne(input: $input) {
            success
        }
    }
`