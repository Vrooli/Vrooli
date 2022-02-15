import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagUpdateMutation = gql`
    ${tagFields}
    mutation tagUpdate($input: TagUpdateInput!) {
        tagUpdate(input: $input) {
            ...tagFields
        }
    }
`