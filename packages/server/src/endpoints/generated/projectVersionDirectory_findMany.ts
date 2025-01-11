export const projectVersionDirectory_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "childOrder": true,
            "isRoot": true,
            "projectVersion": {
                "id": true,
                "complexity": true,
                "isLatest": true,
                "isPrivate": true,
                "versionIndex": true,
                "versionLabel": true,
                "root": {
                    "id": true,
                    "isPrivate": true
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "name": true
                }
            },
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "884059758"
};