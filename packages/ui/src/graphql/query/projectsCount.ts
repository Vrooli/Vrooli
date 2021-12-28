import { gql } from 'graphql-tag';

export const projectsCountQuery = gql`
    query projectsCount {
        projectsCount {
            count
        }
    }
`