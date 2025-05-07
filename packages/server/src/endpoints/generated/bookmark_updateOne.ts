export const bookmark_updateOne = {
    "id": true,
    "list": {
        "id": true,
        "createdAt": true,
        "updatedAt": true,
        "label": true,
        "bookmarksCount": true
    },
    "to": {
        "Comment": {
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
        "Tag": {
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
    },
    "__cacheKey": "-1842251120"
};