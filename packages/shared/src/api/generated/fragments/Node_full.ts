export const Node_full = `fragment Node_full on Node {
id
created_at
updated_at
columnIndex
nodeType
rowIndex
end {
    id
    wasSuccessful
    suggestedNextRoutineVersions {
        id
        complexity
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
    }
}
routineList {
    id
    isOrdered
    isOptional
    items {
        id
        index
        isOptional
        translations {
            id
            language
            description
            name
        }
    }
}
translations {
    id
    language
    description
    name
}
}`;
