export const schedule_createOne = {
    "id": true,
    "publicId": true,
    "createdAt": true,
    "updatedAt": true,
    "startTime": true,
    "endTime": true,
    "timezone": true,
    "exceptions": {
        "id": true,
        "originalStartTime": true,
        "newStartTime": true,
        "newEndTime": true
    },
    "recurrences": {
        "id": true,
        "recurrenceType": true,
        "interval": true,
        "dayOfWeek": true,
        "dayOfMonth": true,
        "month": true,
        "endDate": true
    },
    "meetings": {
        "id": true,
        "publicId": true,
        "createdAt": true,
        "updatedAt": true,
        "openToAnyoneWithInvite": true,
        "showOnTeamProfile": true,
        "team": {
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
        "attendeesCount": true,
        "invitesCount": true,
        "you": {
            "canDelete": true,
            "canInvite": true,
            "canUpdate": true
        },
        "attendees": {
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
        "invites": {
            "id": true,
            "createdAt": true,
            "updatedAt": true,
            "message": true,
            "status": true,
            "you": {
                "canDelete": true,
                "canUpdate": true
            }
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "link": true,
            "name": true
        }
    },
    "runs": {
        "id": true,
        "isPrivate": true,
        "completedComplexity": true,
        "contextSwitches": true,
        "data": true,
        "startedAt": true,
        "timeElapsed": true,
        "completedAt": true,
        "name": true,
        "status": true,
        "ioCount": true,
        "stepsCount": true,
        "wasRunAutomatically": true,
        "resourceVersion": {
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
                "isPrivate": true,
                "publicId": true,
                "createdAt": true,
                "updatedAt": true,
                "bookmarks": true,
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
                }
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
            "versionLabel": true,
            "createdAt": true,
            "updatedAt": true,
            "codeLanguage": true,
            "completedAt": true,
            "timesStarted": true,
            "timesCompleted": true,
            "commentsCount": true,
            "forksCount": true,
            "reportsCount": true,
            "config": true,
            "versionNotes": true,
            "pullRequest": {
                "id": true,
                "publicId": true,
                "createdAt": true,
                "updatedAt": true,
                "closedAt": true,
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
            }
        },
        "team": {
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
        "user": {
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
            "canDelete": true,
            "canUpdate": true,
            "canRead": true
        },
        "lastStep": true,
        "io": {
            "id": true,
            "data": true,
            "nodeInputName": true,
            "nodeName": true
        },
        "steps": {
            "id": true,
            "order": true,
            "contextSwitches": true,
            "startedAt": true,
            "timeElapsed": true,
            "completedAt": true,
            "name": true,
            "status": true,
            "resourceInId": true,
            "resourceVersion": {
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
        }
    },
    "__cacheKey": "-540000699"
};