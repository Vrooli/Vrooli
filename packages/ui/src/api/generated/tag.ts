import gql from 'graphql-tag';

export const findOne = gql`
query tag($input: FindByIdInput!) {
  tag(input: $input) {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
  }
}`;

export const findMany = gql`
query tags($input: TagSearchInput!) {
  tags(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            tag
            stars
            translations {
                id
                language
                description
            }
            you {
                isOwn
                isStarred
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
mutation tagCreate($input: TagCreateInput!) {
  tagCreate(input: $input) {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
  }
}`;

export const update = gql`
mutation tagUpdate($input: TagUpdateInput!) {
  tagUpdate(input: $input) {
    id
    created_at
    tag
    stars
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isStarred
    }
  }
}`;

