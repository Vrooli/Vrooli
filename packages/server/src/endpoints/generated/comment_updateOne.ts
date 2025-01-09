export const comment_updateOne = {
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
        "Post": {
            "id": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
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
        "Question": {
            "id": true,
            "created_at": true,
            "updated_at": true,
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
            "hasAcceptedAnswer": true,
            "isPrivate": true,
            "score": true,
            "bookmarks": true,
            "answersCount": true,
            "commentsCount": true,
            "reportsCount": true,
            "forObject": {
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
            "you": {
                "reaction": true
            }
        },
        "QuestionAnswer": {
            "id": true,
            "created_at": true,
            "updated_at": true,
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
            "score": true,
            "bookmarks": true,
            "isAccepted": true,
            "commentsCount": true
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
    }
};