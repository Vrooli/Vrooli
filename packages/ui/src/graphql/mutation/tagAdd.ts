import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagAddMutation = gql`
    ${tagFields}
    mutation tagAdd($input: TagAddInput!) {
        tagAdd(input: $input) {
            ...tagFields
        }
    }
`