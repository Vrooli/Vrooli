export const ApiVersion_nav = `fragment ApiVersion_nav on ApiVersion {
id
isLatest
isPrivate
versionIndex
versionLabel
root {
    id
    isPrivate
}
translations {
    id
    language
    details
    name
    summary
}
}`;