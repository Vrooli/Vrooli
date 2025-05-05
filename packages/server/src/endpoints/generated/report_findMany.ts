export const report_findMany = {
    "edges": {
        "cursor": true,
        "node": {
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
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "1419805711"
};