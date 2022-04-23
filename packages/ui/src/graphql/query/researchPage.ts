import { gql } from 'graphql-tag';
import { deepRoutineFields, organizationFields, projectFields, routineFields } from 'graphql/fragment';

export const researchPageQuery = gql`
    ${deepRoutineFields}
    ${organizationFields}
    ${projectFields}
    ${routineFields}
    query researchPage {
        researchPage {
            processes {
                ...routineFields
            }
            newlyCompleted {
                ... on Project {
                    ...projectFields
                }
                ... on Routine {
                    ...deepRoutineFields
                }
            }
            needVotes {
                ...projectFields
            }
            needInvestments {
                ...organizationFields
            }
            needMembers {
                ...organizationFields
            }
        }
    }
`