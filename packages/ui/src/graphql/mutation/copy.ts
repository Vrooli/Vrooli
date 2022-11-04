import { gql } from 'graphql-tag';
import { organizationFields, projectFields, routineFields, routineNodeFields, standardFields } from 'graphql/fragment';

export const copyMutation = gql`
    ${organizationFields}
    ${projectFields}
    ${routineFields}
    ${routineNodeFields}
    ${standardFields}
    mutation copy($input: CopyInput!) {
        copy(input: $input) {
            organization {
                ...organizationFields
            }
            project {
                ...projectFields
            }
            routine {
                ...routineFields
            }
            routineNode {
                ...routineNodeFields
            }
            standard {
                ...standardFields
            }
        }
    }
`