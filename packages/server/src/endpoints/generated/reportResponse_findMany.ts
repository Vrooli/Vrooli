export const reportResponse_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "actionSuggested": true,
            "details": true,
            "language": true,
            "report": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "details": true,
                "language": true,
                "reason": true,
                "responsesCount": true,
                "status": true,
                "you": {
                    "canDelete": true,
                    "canRespond": true,
                    "canUpdate": true,
                    "isOwn": true
                }
            },
            "you": {
                "canDelete": true,
                "canUpdate": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    }
};