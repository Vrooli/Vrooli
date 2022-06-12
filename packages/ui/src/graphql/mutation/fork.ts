import { gql } from 'graphql-tag';
import { organizationFields, projectFields, routineFields, standardFields } from 'graphql/fragment';

export const forkMutation = gql`
    ${organizationFields}
    ${projectFields}
    ${routineFields}
    ${standardFields}
    mutation fork($input: ForkInput!) {
        fork(input: $input) {
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