export const unions_projectOrRoutine = {
    "edges": {
        "cursor": true,
        "node": {
            "Project": {
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
                "versions": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "directoriesCount": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "reportsCount": true,
                    "runProjectsCount": true,
                    "simplicity": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "name": true
                    }
                }
            },
            "Routine": {
                "id": true,
                "created_at": true,
                "updated_at": true,
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
                "versions": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "completedAt": true,
                    "isAutomatable": true,
                    "isComplete": true,
                    "isDeleted": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "routineType": true,
                    "simplicity": true,
                    "timesStarted": true,
                    "timesCompleted": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "commentsCount": true,
                    "directoryListingsCount": true,
                    "forksCount": true,
                    "inputsCount": true,
                    "outputsCount": true,
                    "reportsCount": true,
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "instructions": true,
                        "name": true
                    }
                }
            }
        }
    },
    "pageInfo": {
        "hasNextPage": true,
        "endCursorProject": true,
        "endCursorRoutine": true
    },
    "__cacheKey": "-684235021"
};