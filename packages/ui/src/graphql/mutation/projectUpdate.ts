import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectUpdateMutation = gql`
    ${projectFields}
    mutation projectUpdate($input: ProjectInput!) {
        projectUpdate(input: $input) {
            ...projectFields
        }
    }
`