import gql from 'graphql-tag';

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

