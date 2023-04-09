export const Label_common = `fragment Label_common on Label {
id
created_at
updated_at
color
label
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