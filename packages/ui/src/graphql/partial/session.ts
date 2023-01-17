export const sessionFields = ['Wallet', `{
    isLoggedIn
    timeZone
    users {
        id
        handle
        hasPremium
        languages
        name
        theme
    }
}`] as const;