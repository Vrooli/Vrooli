import { gql } from 'graphql-tag';

export const projectDeleteOneMutation = gql`
    mutation projectDeleteOne($input: DeleteOneInput!) {
        projectDeleteOne(input: $input) {
            success
        }
    }
`