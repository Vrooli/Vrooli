export const comment_createOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
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
        "ApiVersion": {
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
                "details": true,
                "name": true,
                "summary": true
            }
        },
        "CodeVersion": {
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
        "Issue": {
            "id": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        },
        "NoteVersion": {
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
        },
        "ProjectVersion": {
            "id": true,
            "complexity": true,
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
                "name": true
            }
        },
        "PullRequest": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "mergedOrRejectedAt": true,
            "status": true
        },
        "RoutineVersion": {
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
            "routineType": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "instructions": true,
                "name": true
            },
            "versionIndex": true,
            "versionLabel": true
        },
        "StandardVersion": {
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
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "text": true
    },
    "__cacheKey": "-1647528212"
};