import { gql } from 'graphql-tag';
import { listOrganizationFields, listProjectFields, listRoutineFields } from 'graphql/fragment';

export const researchPageQuery = gql`
    ${listOrganizationFields}
    ${listProjectFields}
    ${listRoutineFields}
    query researchPage {
        researchPage {
            processes {
                ...listRoutineFields
            }
            newlyCompleted {
                ... on Project {
                    ...listProjectFields
                }
                ... on Routine {
                    ...listRoutineFields
                }
            }
            needVotes {
                ...listProjectFields
            }
            needInvestments {
                ...listProjectFields
            }
            needMembers {
                ...listOrganizationFields
            }
        }
    }
`