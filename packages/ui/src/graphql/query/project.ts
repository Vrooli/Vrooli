import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectQuery = gql`
    ${projectFields}
    query project($input: FindByIdOrHandleInput!) {
        project(input: $input) {
            ...projectFields
        }
    }
`