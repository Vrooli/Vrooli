import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectQuery = gql`
    ${projectFields}
    query project($input: FindByIdInput!) {
        project(input: $input) {
            ...projectFields
        }
    }
`