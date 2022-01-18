import { gql } from 'graphql-tag';

export const standardDeleteOneMutation = gql`
    mutation standardDeleteOne($input: DeleteOneInput!) {
        standardDeleteOne(input: $input) {
            success
        }
    }
`