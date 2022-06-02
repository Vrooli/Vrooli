import { gql } from 'graphql-tag';

export const deleteOneMutation = gql`
    mutation deleteOne($input: DeleteOneInput!) {
        deleteOne(input: $input) {
            success
        }
    }
`