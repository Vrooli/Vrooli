import { gql } from 'graphql-tag';
import { tagFields } from '.';

export const routineFields = gql`
    ${tagFields}
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