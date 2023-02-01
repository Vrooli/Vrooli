export const StandardVersion_nav = `fragment StandardVersion_nav on StandardVersion {
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
    description
    jsonVariable
}
}`;