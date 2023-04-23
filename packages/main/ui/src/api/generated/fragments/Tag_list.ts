export const Tag_list = `fragment Tag_list on Tag {
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
}`;
