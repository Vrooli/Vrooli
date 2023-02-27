import gql from 'graphql-tag';

export const walletFindHandles = gql`
query findHandles($input: FindHandlesInput!) {
  findHandles(input: $input)
}`;

