import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment routineFields on Routine {
        id
        version
        title
        description
        created_at
        isAutomatable
    }
`