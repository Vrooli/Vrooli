import { gql } from 'graphql-tag';

export const routineFields = gql`
    fragment tagFields on Tag {
        id
        description
        tag
    }
    fragment routineFields on Routine {
        id
        version
        title
        description
        created_at
        isAutomatable
        tags {
            ...tagFields
        }
    }
`