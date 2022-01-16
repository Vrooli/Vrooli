import { gql } from 'graphql-tag';
import { voteFields } from 'graphql/fragment';

export const voteAddMutation = gql`
    ${voteFields}
    mutation voteAdd($input: VoteInput!) {
        voteAdd(input: $input) {
            ...voteFields
        }
    }
`