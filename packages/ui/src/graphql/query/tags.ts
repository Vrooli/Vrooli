import { gql } from 'graphql-tag';
import { tagFields } from 'graphql/fragment';

export const tagsQuery = gql`
    ${tagFields}
    query tags($input: TagsQueryInput!) {
        tags(input: $input) {
            ...tagFields
        }
    }
`