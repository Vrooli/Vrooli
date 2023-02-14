import gql from 'graphql-tag';

export const tagFindOne = gql`
query tag($input: FindByIdInput!) {
  tag(input: $input) {
    id
    created_at
    tag
    bookmarks
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isBookmarked
    }
  }
}`;

export const tagFindMany = gql`
query tags($input: TagSearchInput!) {
  tags(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            tag
            bookmarks
            translations {
                id
                language
                description
            }
            you {
                isOwn
                isBookmarked
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const tagCreate = gql`
mutation tagCreate($input: TagCreateInput!) {
  tagCreate(input: $input) {
    id
    created_at
    tag
    bookmarks
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isBookmarked
    }
  }
}`;

export const tagUpdate = gql`
mutation tagUpdate($input: TagUpdateInput!) {
  tagUpdate(input: $input) {
    id
    created_at
    tag
    bookmarks
    translations {
        id
        language
        description
    }
    you {
        isOwn
        isBookmarked
    }
  }
}`;

