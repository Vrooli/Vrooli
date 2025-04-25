export const reminder_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "description": true,
            "dueDate": true,
            "index": true,
            "isComplete": true,
            "reminderItems": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "name": true,
                "description": true,
                "dueDate": true,
                "index": true,
                "isComplete": true
            },
            "reminderList": {
                "id": true,
                "created_at": true,
                "updated_at": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "1430826308"
};