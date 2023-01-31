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

export const resourceFindMany = gql`
query resources($input: ResourceSearchInput!) {
  resources(input: $input) {
    edges {
        cursor
        node {
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
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

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

