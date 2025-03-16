export const issue_findOne = {
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
    "labels": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "color": true,
        "label": true,
        "you": {
            "canDelete": true,
            "canUpdate": true
        }
    },
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
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "isBotDepictingPerson": true,
        "name": true,
        "profileImage": true
    },
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
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
    },
    "__cacheKey": "-1716964463"
};