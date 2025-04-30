export const reportResponse_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "actionSuggested": true,
            "details": true,
            "language": true,
            "report": {
                "id": true,
                "publicId": true,
                "createdAt": true,
                "updatedAt": true,
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
    },
    "__cacheKey": "360802519"
};