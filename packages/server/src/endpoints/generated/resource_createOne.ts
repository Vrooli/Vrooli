export const resource_createOne = {
    "id": true,
    "publicId": true,
    "createdAt": true,
    "updatedAt": true,
    "bookmarks": true,
    "isInternal": true,
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
    "permissions": true,
    "resourceType": true,
    "score": true,
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
    "transfersCount": true,
    "views": true,
    "you": {
        "canComment": true,
        "canDelete": true,
        "canBookmark": true,
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
    },
    "versions": {
        "id": true,
        "createdAt": true,
        "updatedAt": true,
        "codeLanguage": true,
        "completedAt": true,
        "isAutomatable": true,
        "isComplete": true,
        "isDeleted": true,
        "isLatest": true,
        "isPrivate": true,
        "resourceSubType": true,
        "simplicity": true,
        "timesStarted": true,
        "timesCompleted": true,
        "versionIndex": true,
        "versionLabel": true,
        "commentsCount": true,
        "forksCount": true,
        "reportsCount": true,
        "you": {
            "canComment": true,
            "canCopy": true,
            "canDelete": true,
            "canBookmark": true,
            "canReport": true,
            "canRun": true,
            "canUpdate": true,
            "canRead": true,
            "canReact": true
        },
        "config": true,
        "versionNotes": true,
        "pullRequest": {
            "id": true,
            "publicId": true,
            "createdAt": true,
            "updatedAt": true,
            "mergedOrRejectedAt": true,
            "commentsCount": true,
            "status": true,
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
        "relatedVersions": {
            "id": true,
            "toVersion": {
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
            "description": true,
            "details": true,
            "instructions": true,
            "name": true
        }
    },
    "__cacheKey": "1133179917"
};