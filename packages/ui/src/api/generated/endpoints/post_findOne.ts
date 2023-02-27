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

