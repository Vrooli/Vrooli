import { gql } from 'graphql-tag';

export const reportDeleteOneMutation = gql`
    mutation reportDeleteOne($input: DeleteOneInput!) {
        reportDeleteOne(input: $input) {
            success
        }
    }
`