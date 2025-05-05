export const memberInvite_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "message": true,
            "status": true,
            "willBeAdmin": true,
            "willHavePermissions": true,
            "team": {
                "id": true,
                "bannerImage": true,
                "handle": true,
                "profileImage": true,
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
            },
            "user": {
                "id": true,
                "createdAt": true,
                "updatedAt": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "isBotDepictingPerson": true,
                "name": true,
                "profileImage": true
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
    "__cacheKey": "1536690302"
};