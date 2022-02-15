import { gql } from 'graphql-tag';

export const organizationDeleteOneMutation = gql`
    mutation organizationDeleteOne($input: DeleteOneInput!) {
        organizationDeleteOne(input: $input) {
            success
        }
    }
`