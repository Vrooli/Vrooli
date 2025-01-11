export const post_findOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "commentsCount": true,
    "repostsCount": true,
    "score": true,
    "bookmarks": true,
    "views": true,
    "resourceList": {
        "id": true,
        "created_at": true,
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
        },
        "resources": {
            "id": true,
            "index": true,
            "link": true,
            "usedFor": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
    },
    "__cacheKey": "-482597638"
};