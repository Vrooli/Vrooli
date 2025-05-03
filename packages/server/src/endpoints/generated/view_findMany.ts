export const view_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "to": {
                "Issue": {
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
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "name": true
                    }
                },
                "Resource": {
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
                        "translations": {
                            "id": true,
                            "language": true,
                            "description": true,
                            "details": true,
                            "instructions": true,
                            "name": true
                        }
                    }
                },
                "Team": {
                    "id": true,
                    "publicId": true,
                    "bannerImage": true,
                    "handle": true,
                    "createdAt": true,
                    "updatedAt": true,
                    "isOpenToNewMembers": true,
                    "isPrivate": true,
                    "commentsCount": true,
                    "membersCount": true,
                    "profileImage": true,
                    "reportsCount": true,
                    "bookmarks": true,
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
                            "createdAt": true,
                            "updatedAt": true,
                            "isAdmin": true,
                            "permissions": true
                        }
                    }
                },
                "User": {
                    "id": true,
                    "publicId": true,
                    "createdAt": true,
                    "updatedAt": true,
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
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "1929325547"
};