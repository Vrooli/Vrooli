import { gql } from 'graphql-tag';
import { nodeFields, organizationFields, projectFields, routineFields, standardFields } from 'graphql/fragment';

export const copyMutation = gql`
    ${nodeFields}
    ${organizationFields}
    ${projectFields}
    ${routineFields}
    ${standardFields}
    mutation copy($input: CopyInput!) {
        copy(input: $input) {
            node {
                ...nodeFields
            }
            organization {
                ...organizationFields
            }
            project {
                ...projectFields
            }
            routine {
                ...routineFields
            }
            standard {
                ...standardFields
            }
        }
    }
`