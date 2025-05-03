export const reminder_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "name": true,
            "description": true,
            "dueDate": true,
            "index": true,
            "isComplete": true,
            "reminderItems": {
                "id": true,
                "createdAt": true,
                "updatedAt": true,
                "name": true,
                "description": true,
                "dueDate": true,
                "index": true,
                "isComplete": true
            },
            "reminderList": {
                "id": true,
                "createdAt": true,
                "updatedAt": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "1292331446"
};