export const notificationSubscription_findMany = {
    "edges": {
        "cursor": true,
        "node": {
            "id": true,
            "createdAt": true,
            "silent": true,
            "object": {
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
                    },
                    "translations": {
                        "id": true,
                        "language": true,
                        "text": true
                    }
                },
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
                "Meeting": {
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
                    "schedule": {
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
                            "lastStep": true
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
                "PullRequest": {
                    "id": true,
                    "publicId": true,
                    "createdAt": true,
                    "updatedAt": true,
                    "closedAt": true,
                    "commentsCount": true,
                    "status": true,
                    "from": {
                        "ResourceVersion": {
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
                            "root": {
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
                        }
                    },
                    "to": {
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
                        }
                    },
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
                "Report": {
                    "id": true,
                    "publicId": true,
                    "createdAt": true,
                    "updatedAt": true,
                    "details": true,
                    "language": true,
                    "reason": true,
                    "responsesCount": true,
                    "status": true,
                    "you": {
                        "canDelete": true,
                        "canRespond": true,
                        "canUpdate": true,
                        "isOwn": true
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
                }
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    },
    "__cacheKey": "1160119365"
};