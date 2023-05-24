export const ProjectVersion_nav = `fragment ProjectVersion_nav on ProjectVersion {
id
complexity
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
}
}`;
