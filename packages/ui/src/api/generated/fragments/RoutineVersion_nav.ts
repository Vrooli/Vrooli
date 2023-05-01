export const RoutineVersion_nav = `fragment RoutineVersion_nav on RoutineVersion {
id
isAutomatable
isComplete
isDeleted
isLatest
isPrivate
root {
    id
    isInternal
    isPrivate
}
translations {
    id
    language
    description
    instructions
    name
}
versionIndex
versionLabel
}`;
