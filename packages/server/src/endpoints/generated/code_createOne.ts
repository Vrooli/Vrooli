export const code_createOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "isPrivate": true,
    "issuesCount": true,
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
    "questionsCount": true,
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
            "jsonVariable": true,
            "name": true
        }
    },
    "versions": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "isComplete": true,
        "isDeleted": true,
        "isLatest": true,
        "isPrivate": true,
        "codeLanguage": true,
        "codeType": true,
        "default": true,
        "versionIndex": true,
        "versionLabel": true,
        "calledByRoutineVersionsCount": true,
        "commentsCount": true,
        "directoryListingsCount": true,
        "forksCount": true,
        "reportsCount": true,
        "you": {
            "canComment": true,
            "canCopy": true,
            "canDelete": true,
            "canReport": true,
            "canUpdate": true,
            "canUse": true,
            "canRead": true
        },
        "content": true,
        "versionNotes": true,
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
        "resourceList": {
            "id": true,
            "created_at": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            },
            "resources": {
                "id": true,
                "index": true,
                "link": true,
                "usedFor": true,
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "name": true
                }
            }
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "jsonVariable": true,
            "name": true
        }
    }
};