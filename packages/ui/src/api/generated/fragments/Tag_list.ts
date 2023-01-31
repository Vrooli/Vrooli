export const Tag_list = `fragment Tag_list on Tag {
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
}`;