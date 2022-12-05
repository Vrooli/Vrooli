import { gql } from 'graphql-tag';
import { listOrganizationFields, listProjectFields, listRoutineFields } from 'graphql/fragment';

export const researchQuery = gql`
    ${listOrganizationFields}
    ${listProjectFields}
    ${listRoutineFields}
    query research {
        research {
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