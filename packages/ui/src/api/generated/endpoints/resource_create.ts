import gql from 'graphql-tag';

export const resourceCreate = gql`
mutation resourceCreate($input: ResourceCreateInput!) {
  resourceCreate(input: $input) {
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

