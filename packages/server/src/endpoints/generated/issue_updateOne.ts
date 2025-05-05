export const issue_updateOne = {
    "id": true,
    "publicId": true,
    "createdAt": true,
    "updatedAt": true,
    "closedAt": true,
    "referencedVersionId": true,
    "status": true,
    "to": {
        "Resource": {
            "id": true,
            "isInternal": true,
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
                    "createdAt": true,
                    "updatedAt": true,
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
    "closedBy": {
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
    "createdBy": {
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
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
    },
    "__cacheKey": "-1252340569"
};