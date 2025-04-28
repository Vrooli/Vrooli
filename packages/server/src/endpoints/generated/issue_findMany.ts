export const issue_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "closedAt": true,
            "referencedVersionId": true,
            "status": true,
            "to": {
                "Api": {
                    "id": true,
                    "isPrivate": true
                },
                "Code": {
                    "id": true,
                    "isPrivate": true
                },
                "Note": {
                    "id": true,
                    "isPrivate": true
                },
                "Project": {
                    "id": true,
                    "isPrivate": true
                },
                "Routine": {
                    "id": true,
                    "isInternal": true,
                    "isPrivate": true
                },
                "Standard": {
                    "id": true,
                    "isPrivate": true
                },
                "Team": {
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
                }
            },
            "commentsCount": true,
            "reportsCount": true,
            "score": true,
            "bookmarks": true,
            "views": true,
            "you": {
                "canComment": true,
                "canDelete": true,
                "canBookmark": true,
                "canReport": true,
                "canUpdate": true,
                "canRead": true,
                "canReact": true,
                "isBookmarked": true,
                "reaction": true
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
    "__cacheKey": "307013224"
};