export const User_list = `fragment User_list on User {
translations {
    id
    language
    bio
}
id
created_at
handle
isBot
name
bookmarks
reportsReceivedCount
you {
    canDelete
    canReport
    canUpdate
    isBookmarked
    isViewed
}
}`;
