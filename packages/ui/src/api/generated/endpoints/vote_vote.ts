import gql from 'graphql-tag';

export const voteVote = gql`
mutation vote($input: VoteInput!) {
  vote(input: $input) {
    success
  }
}`;

