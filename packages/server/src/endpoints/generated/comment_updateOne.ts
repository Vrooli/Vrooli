export const comment_updateOne = {
    "id": true,
    "createdAt": true,
    "updatedAt": true,
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
                    "createdAt": true,
                    "updatedAt": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        },
        "User": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true
        }
    },
    "score": true,
    "bookmarks": true,
    "reportsCount": true,
    "you": {
        "canDelete": true,
        "canBookmark": true,
        "canReply": true,
        "canReport": true,
        "canUpdate": true,
        "canReact": true,
        "isBookmarked": true,
        "reaction": true
    },
    "commentedOn": {
        "Issue": {
            "id": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        },
        "PullRequest": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "closedAt": true,
            "status": true
        },
        "ResourceVersion": {
            "id": true,
            "complexity": true,
            "isAutomatable": true,
            "isComplete": true,
            "isDeleted": true,
            "isLatest": true,
            "isPrivate": true,
            "root": {
                "id": true,
                "isInternal": true,
                "isPrivate": true
            },
            "resourceSubType": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "details": true,
                "instructions": true,
                "name": true
            },
            "versionIndex": true,
            "versionLabel": true
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "text": true
    },
    "__cacheKey": "-832183216"
};