import { gql } from 'graphql-tag';

export const tagsCountQuery = gql`
    query tagsCount($input: TagCountInput!) {
        tagsCount(input: $input)
    }
`