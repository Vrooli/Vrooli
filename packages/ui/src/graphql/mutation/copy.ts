import { gql } from 'graphql-tag';
import { organizationFields, projectFields, routineFields, standardFields } from 'graphql/fragment';

export const copyMutation = gql`
    ${organizationFields}
    ${projectFields}
    ${routineFields}
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
            standard {
                ...standardFields
            }
        }
    }
`