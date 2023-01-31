import gql from 'graphql-tag';

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

export const resourceListFindMany = gql`
query resourceLists($input: ResourceListSearchInput!) {
  resourceLists(input: $input) {
    edges {
        cursor
        node {
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
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

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

