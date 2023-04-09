import gql from 'graphql-tag';

export const pushDeviceFindMany = gql`
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

