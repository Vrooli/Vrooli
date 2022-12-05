import { gql } from 'graphql-tag';
import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/fragment';

export const popularQuery = gql`
    ${listOrganizationFields}
    ${listProjectFields}
    ${listRoutineFields}
    ${listStandardFields}
    ${listUserFields}
    query popular($input: PopularInput!) {
        popular(input: $input) {
            organizations {
                ...listOrganizationFields
            }
            projects {
                ...listProjectFields
            }
            routines {
                ...listRoutineFields
            }
            standards {
                ...listStandardFields
            }
            users {
                ...listUserFields
            }
        }
    }
`