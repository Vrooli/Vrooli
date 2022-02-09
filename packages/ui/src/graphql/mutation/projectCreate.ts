import { gql } from 'graphql-tag';
import { projectFields } from 'graphql/fragment';

export const projectCreateMutation = gql`
    ${projectFields}
    mutation projectCreate($input: ProjectCreateInput!) {
        projectCreate(input: $input) {
            ...projectFields
        }
    }
`