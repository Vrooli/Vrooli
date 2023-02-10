export const NoteVersion_nav = `fragment NoteVersion_nav on NoteVersion {
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
    name
    text
}
}`;