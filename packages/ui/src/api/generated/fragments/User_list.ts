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
reportsCount
you {
    canDelete
    canEdit
    canReport
    isStarred
    isViewed
}
}`;