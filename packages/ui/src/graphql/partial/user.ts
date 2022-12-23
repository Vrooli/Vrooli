export const userNameFields = ['User', `{
    id
    name
    handle
}`] as const;
export const listUserFields = ['User', `{
    id
    handle
    name
    stars
    isStarred
    reportsCount
    translations {
        id
        language
        bio
    }
}`] as const;
export const userFields = ['User', `{
    id
    handle
    name
    created_at
    stars
    isStarred
    reportsCount
    translations {
        id
        language
        bio
    }
}`] as const;