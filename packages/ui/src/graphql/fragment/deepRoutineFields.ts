import { gql } from 'graphql-tag';
import { nodeFields, routineFields, standardFields } from '.';

export const deepRoutineFields = gql`
    ${nodeFields}
    ${routineFields}
    ${standardFields}
    fragment deepRoutineFields on Routine {
        ...routineFields
        instructions
        inputs {
            routine {
                ...routineFields
            }
            standard {
                ...standardFields
            }
        }
        outputs {
            routine {
                ...routineFields
            }
            standard {
                ...standardFields
            }
        }
        nodes {
            ...nodeFields
        }
    }
`