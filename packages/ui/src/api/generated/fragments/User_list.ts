export const User_list = `fragment User_list on User {
translations {
    id
    language
    bio
}
id
created_at
handle
name
stars
reportsReceivedCount
you {
    canDelete
    canReport
    canUpdate
    isStarred
    isViewed
}
}`;