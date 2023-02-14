import gql from 'graphql-tag';

export const postFindOne = gql`
query post($input: FindByIdInput!) {
  post(input: $input) {
    resourceList {
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
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    commentsCount
    repostsCount
    score
    bookmarks
    views
  }
}`;

export const postFindMany = gql`
query posts($input: PostSearchInput!) {
  posts(input: $input) {
    edges {
        cursor
        node {
            resourceList {
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
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            commentsCount
            repostsCount
            score
            bookmarks
            views
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const postCreate = gql`
mutation postCreate($input: PostCreateInput!) {
  postCreate(input: $input) {
    resourceList {
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
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    commentsCount
    repostsCount
    score
    bookmarks
    views
  }
}`;

export const postUpdate = gql`
mutation postUpdate($input: PostUpdateInput!) {
  postUpdate(input: $input) {
    resourceList {
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
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    commentsCount
    repostsCount
    score
    bookmarks
    views
  }
}`;

