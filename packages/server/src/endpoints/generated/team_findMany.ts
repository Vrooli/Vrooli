export const team_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "publicId": true,
            "bannerImage": true,
            "handle": true,
            "createdAt": true,
            "updatedAt": true,
            "isOpenToNewMembers": true,
            "isPrivate": true,
            "commentsCount": true,
            "membersCount": true,
            "profileImage": true,
            "reportsCount": true,
            "bookmarks": true,
            "tags": {
                "id": true,
                "createdAt": true,
                "tag": true,
                "bookmarks": true,
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true
                },
                "you": {
                    "isOwn": true,
                    "isBookmarked": true
                }
            },
            "translations": {
                "id": true,
                "language": true,
                "bio": true,
                "name": true
            },
            "you": {
                "canAddMembers": true,
                "canDelete": true,
                "canBookmark": true,
                "canReport": true,
                "canUpdate": true,
                "canRead": true,
                "isBookmarked": true,
                "isViewed": true,
                "yourMembership": {
                    "id": true,
                    "createdAt": true,
                    "updatedAt": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "-970363431"
};