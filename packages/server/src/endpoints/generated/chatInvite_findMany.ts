export const chatInvite_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "message": true,
            "status": true,
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
            },
            "chat": {
                "id": true,
                "publicId": true,
                "createdAt": true,
                "updatedAt": true,
                "openToAnyoneWithInvite": true,
                "participants": {
                    "id": true,
                    "createdAt": true,
                    "updatedAt": true,
                    "user": {
                        "id": true,
                        "publicId": true,
                        "createdAt": true,
                        "updatedAt": true,
                        "bannerImage": true,
                        "handle": true,
                        "isBot": true,
                        "isBotDepictingPerson": true,
                        "name": true,
                        "profileImage": true,
                        "bookmarks": true,
                        "reportsReceivedCount": true,
                        "you": {
                            "canDelete": true,
                            "canReport": true,
                            "canUpdate": true,
                            "isBookmarked": true,
                            "isViewed": true
                        },
                        "translations": {
                            "id": true,
                            "language": true,
                            "bio": true
                        }
                    }
                },
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
                "participantsCount": true,
                "invitesCount": true,
                "you": {
                    "canDelete": true,
                    "canInvite": true,
                    "canUpdate": true
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "name": true
                }
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "641208870"
};