import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectAddMutation = gql`
    ${projectFields}
    mutation projectAdd($input: ProjectInput!) {
        projectAdd(input: $input) {
            ...projectFields
        }
    }
`