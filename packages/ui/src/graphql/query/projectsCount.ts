import { gql } from 'graphql-tag';

export const projectsCountQuery = gql`
    query projectsCount($input: ProjectCountInput!) {
        projectsCount(input: $input)
    }
`