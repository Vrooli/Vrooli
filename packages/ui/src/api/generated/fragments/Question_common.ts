export const Question_common = `fragment Question_common on Question {
id
created_at
updated_at
createdBy {
    id
    name
    handle
}
hasAcceptedAnswer
isPrivate
score
bookmarks
answersCount
commentsCount
reportsCount
forObject {
    ... on Api {
        ...Api_nav
    }
    ... on Note {
        ...Note_nav
    }
    ... on Organization {
        ...Organization_nav
    }
    ... on Project {
        ...Project_nav
    }
    ... on Routine {
        ...Routine_nav
    }
    ... on SmartContract {
        ...SmartContract_nav
    }
    ... on Standard {
        ...Standard_nav
    }
}
tags {
    ...Tag_list
}
you {
    reaction
}
}`;