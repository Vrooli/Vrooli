import { gql } from 'graphql-tag';
import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/fragment';

export const homePageQuery = gql`
    ${listOrganizationFields}
    ${listProjectFields}
    ${listRoutineFields}
    ${listStandardFields}
    ${listUserFields}
    query homePage($input: HomePageInput!) {
        homePage(input: $input) {
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