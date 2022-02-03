import { gql } from 'graphql-tag';

export const emailDeleteOneMutation = gql`
    mutation emailDeleteOne($input: DeleteOneInput!) {
        emailDeleteOne(input: $input) {
            success
        }
    }
`