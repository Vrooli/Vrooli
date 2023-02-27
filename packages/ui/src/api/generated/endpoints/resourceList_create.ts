import gql from 'graphql-tag';

export const resourceListCreate = gql`
mutation resourceListCreate($input: ResourceListCreateInput!) {
  resourceListCreate(input: $input) {
    id
    created_at
    translations {
        id
        language
        description
        name
    }
    resources {
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
  }
}`;

