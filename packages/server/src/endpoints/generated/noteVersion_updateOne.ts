export const noteVersion_updateOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "isLatest": true,
    "isPrivate": true,
    "reportsCount": true,
    "versionIndex": true,
    "versionLabel": true,
    "you": {
        "canComment": true,
        "canCopy": true,
        "canDelete": true,
        "canReport": true,
        "canUpdate": true,
        "canUse": true,
        "canRead": true
    },
    "pullRequest": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "mergedOrRejectedAt": true,
        "commentsCount": true,
        "status": true,
        "createdBy": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true
        },
        "you": {
            "canComment": true,
            "canDelete": true,
            "canReport": true,
            "canUpdate": true
        },
        "translations": {
            "id": true,
            "language": true,
            "text": true
        }
    },
    "root": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "isPrivate": true,
        "issuesCount": true,
        "owner": {
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
            },
            "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "isBotDepictingPerson": true,
                "name": true,
                "profileImage": true
            }
        },
        "permissions": true,
        "score": true,
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
        "transfersCount": true,
        "views": true,
        "you": {
            "canDelete": true,
            "canBookmark": true,
            "canTransfer": true,
            "canUpdate": true,
            "canRead": true,
            "canReact": true,
            "isBookmarked": true,
            "isViewed": true,
            "reaction": true
        },
        "versionsCount": true,
        "parent": {
            "id": true,
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
                "name": true,
                "pages": {
                    "id": true,
                    "pageIndex": true,
                    "text": true
                }
            }
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true,
        "pages": {
            "id": true,
            "pageIndex": true,
            "text": true
        }
    },
    "versionNotes": true,
    "__cacheKey": "1756774086"
};