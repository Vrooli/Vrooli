export const member_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "isAdmin": true,
            "permissions": true,
            "roles": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "name": true,
                "permissions": true,
                "membersCount": true,
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
                            "created_at": true,
                            "updated_at": true,
                            "isAdmin": true,
                            "permissions": true
                        }
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true
                }
            },
            "you": {
                "canDelete": true,
                "canUpdate": true
            },
            "team": {
                "id": true,
                "bannerImage": true,
                "handle": true,
                "created_at": true,
                "updated_at": true,
                "isOpenToNewMembers": true,
                "isPrivate": true,
                "commentsCount": true,
                "membersCount": true,
                "profileImage": true,
                "reportsCount": true,
                "bookmarks": true,
                "tags": {
                    "id": true,
                    "created_at": true,
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
                        "created_at": true,
                        "updated_at": true,
                        "isAdmin": true,
                        "permissions": true
                    }
                }
            },
            "user": {
                "id": true,
                "created_at": true,
                "updated_at": true,
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
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    }
};