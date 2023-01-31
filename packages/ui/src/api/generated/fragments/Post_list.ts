export const Post_list = `fragment Post_list on Post {
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
stars
views
}`;