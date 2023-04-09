import gql from 'graphql-tag';

export const resourceUpdate = gql`
mutation resourceUpdate($input: ResourceUpdateInput!) {
  resourceUpdate(input: $input) {
    id
    index
    link
    usedFor
    translations {
        id
        language
        description
        name
    }
  }
}`;

