import gql from 'graphql-tag';

export const findMany = gql`
query pushDevices($input: PushDeviceSearchInput!) {
  pushDevices(input: $input) {
    edges {
        cursor
        node {
            id
            expires
            name
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const create = gql`
mutation pushDeviceCreate($input: PushDeviceCreateInput!) {
  pushDeviceCreate(input: $input) {
    id
    expires
    name
  }
}`;

export const update = gql`
mutation pushDeviceUpdate($input: PushDeviceUpdateInput!) {
  pushDeviceUpdate(input: $input) {
    id
    expires
    name
  }
}`;

