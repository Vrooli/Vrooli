import gql from 'graphql-tag';

export const reactionReact = gql`
mutation react($input: ReactInput!) {
  react(input: $input) {
    success
  }
}`;

