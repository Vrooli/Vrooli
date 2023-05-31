import gql from "graphql-tag";

export const resourceListUpdate = gql`
mutation resourceListUpdate($input: ResourceListUpdateInput!) {
  resourceListUpdate(input: $input) {
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

