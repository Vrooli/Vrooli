import gql from 'graphql-tag';

export const findOne = gql`
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

export const findMany = gql`
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

export const create = gql`
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

export const update = gql`
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

