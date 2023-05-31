export const ResourceList_full = `fragment ResourceList_full on ResourceList {
id
created_at
translations {
    id
    language
    description
    name
}
resources {
    id
    index
    link
    usedFor
    translations {
        id
        language
        description
        name
    }
}
}`;
