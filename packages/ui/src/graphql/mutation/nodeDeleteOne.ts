import { gql } from 'graphql-tag';

export const nodeDeleteOneMutation = gql`
    mutation nodeDeleteOne($input: DeleteOneInput!) {
        nodeDeleteOne(input: $input) {
            success
        }
    }
`