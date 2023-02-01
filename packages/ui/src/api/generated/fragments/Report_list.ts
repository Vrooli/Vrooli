export const Report_list = `fragment Report_list on Report {
id
created_at
updated_at
details
language
reason
responsesCount
you {
    canDelete
    canEdit
    canRespond
}
}`;