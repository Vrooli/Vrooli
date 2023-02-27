import gql from 'graphql-tag';

export const resourceFindOne = gql`
query resource($input: FindByIdInput!) {
  resource(input: $input) {
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

