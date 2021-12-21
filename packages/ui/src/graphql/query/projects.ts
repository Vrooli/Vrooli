import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const projectsQuery = gql`
    ${resourceFields}
    query projects(
        $input: [ProjectsQueryInput!]!
    ) {
        projects(
            input: $input
        ) {
            id
            name
            description
            resources {
                ...resourceFields
            }
        }
    }
`