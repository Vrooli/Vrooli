export const notification_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "category": true,
            "isRead": true,
            "title": true,
            "description": true,
            "link": true,
            "imgLink": true
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    }
};