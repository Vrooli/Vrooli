export const Label_full = `fragment Label_full on Label {
apisCount
focusModesCount
issuesCount
meetingsCount
notesCount
projectsCount
routinesCount
schedulesCount
smartContractsCount
standardsCount
id
created_at
updated_at
color
owner {
    ... on Organization {
        ...Organization_nav
    }
    ... on User {
        ...User_nav
    }
}
you {
    canDelete
    canUpdate
}
}`;