import gql from "graphql-tag";

export const resourceListFindOne = gql`
query resourceList($input: FindByIdInput!) {
  resourceList(input: $input) {
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

