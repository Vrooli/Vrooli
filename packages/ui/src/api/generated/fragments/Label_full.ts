export const Label_full = `fragment Label_full on Label {
apisCount
issuesCount
meetingsCount
notesCount
projectsCount
routinesCount
runProjectSchedulesCount
runRoutineSchedulesCount
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
    canEdit
}
}`;