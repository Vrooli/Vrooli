import { gql } from 'graphql-tag';
import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/fragment';

export const forYouPageQuery = gql`
    ${listOrganizationFields}
    ${listProjectFields}
    ${listRoutineFields}
    ${listStandardFields}
    ${listUserFields}
    query forYouPage($input: ForYouPageInput!) {
        forYouPage(input: $input) {
            activeRoutines {
                ...listRoutineFields
            }
            completedRoutines {
                ...listRoutineFields
            }
            recent {
                ... on Project {
                    ...listProjectFields
                }
                ... on Organization {
                    ...listOrganizationFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
                ... on Standard {
                    ...listStandardFields
                }
                ... on User {
                    ...listUserFields
                }
            }
            starred {
                ... on Project {
                    ...listProjectFields
                }
                ... on Organization {
                    ...listOrganizationFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
                ... on Standard {
                    ...listStandardFields
                }
                ... on User {
                    ...listUserFields
                }
            }
        }
    }
`