import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagCreateMutation = gql`
    ${tagFields}
    mutation tagCreate($input: TagCreateInput!) {
        tagCreate(input: $input) {
            ...tagFields
        }
    }
`