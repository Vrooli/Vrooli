export const ChatMessage_list = `fragment ChatMessage_list on ChatMessage {
translations {
    id
    language
    text
}
id
created_at
updated_at
user {
    id
    isBot
    name
    handle
}
score
reportsCount
you {
    canDelete
    canReply
    canReport
    canUpdate
    canReact
    reaction
}
}`;
