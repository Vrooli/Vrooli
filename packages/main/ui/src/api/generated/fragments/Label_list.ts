export const Label_list = `fragment Label_list on Label {
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
