export const schedule_findOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
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
    "labels": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "color": true,
        "label": true,
        "you": {
            "canDelete": true,
            "canUpdate": true
        },
        "apisCount": true,
        "codesCount": true,
        "focusModesCount": true,
        "issuesCount": true,
        "meetingsCount": true,
        "notesCount": true,
        "projectsCount": true,
        "routinesCount": true,
        "schedulesCount": true,
        "standardsCount": true
    },
    "focusModes": {
        "id": true,
        "name": true,
        "description": true,
        "you": {
            "canDelete": true,
            "canRead": true,
            "canUpdate": true
        },
        "filters": {
            "id": true,
            "filterType": true,
            "tag": {
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
            }
        },
        "labels": {
            "id": true,
            "color": true,
            "label": true
        },
        "reminderList": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "reminders": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "name": true,
                "description": true,
                "dueDate": true,
                "index": true,
                "isComplete": true,
                "reminderItems": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "name": true,
                    "description": true,
                    "dueDate": true,
                    "index": true,
                    "isComplete": true
                }
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
        }
    },
    "meetings": {
        "id": true,
        "created_at": true,
        "updated_at": true,
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
                    "created_at": true,
                    "updated_at": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        },
        "restrictedToRoles": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "permissions": true,
            "membersCount": true,
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
                        "created_at": true,
                        "updated_at": true,
                        "isAdmin": true,
                        "permissions": true
                    }
                }
            },
            "translations": {
                "id": true,
                "language": true,
                "description": true
            },
            "members": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "isAdmin": true,
                "permissions": true,
                "roles": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "name": true,
                    "permissions": true,
                    "membersCount": true,
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
                                "created_at": true,
                                "updated_at": true,
                                "isAdmin": true,
                                "permissions": true
                            }
                        }
                    },
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true
                    }
                },
                "you": {
                    "canDelete": true,
                    "canUpdate": true
                },
                "user": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
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
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true
        },
        "invites": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "message": true,
            "status": true,
            "you": {
                "canDelete": true,
                "canUpdate": true
            }
        },
        "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "you": {
                "canDelete": true,
                "canUpdate": true
            },
            "apisCount": true,
            "codesCount": true,
            "focusModesCount": true,
            "issuesCount": true,
            "meetingsCount": true,
            "notesCount": true,
            "projectsCount": true,
            "routinesCount": true,
            "schedulesCount": true,
            "standardsCount": true
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "link": true,
            "name": true
        }
    },
    "runProjects": {
        "id": true,
        "isPrivate": true,
        "completedComplexity": true,
        "contextSwitches": true,
        "startedAt": true,
        "timeElapsed": true,
        "completedAt": true,
        "name": true,
        "projectVersion": {
            "id": true,
            "complexity": true,
            "isLatest": true,
            "isPrivate": true,
            "versionIndex": true,
            "versionLabel": true,
            "root": {
                "id": true,
                "isPrivate": true,
                "created_at": true,
                "updated_at": true,
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
                }
            },
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            },
            "created_at": true,
            "updated_at": true,
            "directoriesCount": true,
            "reportsCount": true,
            "runProjectsCount": true,
            "simplicity": true
        },
        "status": true,
        "stepsCount": true,
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
                    "created_at": true,
                    "updated_at": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        },
        "user": {
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
            "canDelete": true,
            "canUpdate": true,
            "canRead": true
        },
        "lastStep": true,
        "steps": {
            "id": true,
            "order": true,
            "contextSwitches": true,
            "startedAt": true,
            "timeElapsed": true,
            "completedAt": true,
            "name": true,
            "status": true,
            "step": true,
            "directory": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "childOrder": true,
                "isRoot": true,
                "projectVersion": {
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
                }
            }
        }
    },
    "runRoutines": {
        "id": true,
        "isPrivate": true,
        "completedComplexity": true,
        "contextSwitches": true,
        "startedAt": true,
        "timeElapsed": true,
        "completedAt": true,
        "name": true,
        "status": true,
        "inputsCount": true,
        "outputsCount": true,
        "stepsCount": true,
        "wasRunAutomatically": true,
        "routineVersion": {
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
                "created_at": true,
                "updated_at": true,
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
                }
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
            "versionLabel": true,
            "created_at": true,
            "updated_at": true,
            "completedAt": true,
            "simplicity": true,
            "timesStarted": true,
            "timesCompleted": true,
            "commentsCount": true,
            "directoryListingsCount": true,
            "forksCount": true,
            "inputsCount": true,
            "nodesCount": true,
            "nodeLinksCount": true,
            "outputsCount": true,
            "reportsCount": true,
            "configCallData": true,
            "configFormInput": true,
            "configFormOutput": true,
            "versionNotes": true,
            "apiVersion": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "callLink": true,
                "commentsCount": true,
                "documentationLink": true,
                "forksCount": true,
                "isLatest": true,
                "isPrivate": true,
                "reportsCount": true,
                "versionIndex": true,
                "versionLabel": true,
                "you": {
                    "canComment": true,
                    "canCopy": true,
                    "canDelete": true,
                    "canReport": true,
                    "canUpdate": true,
                    "canUse": true,
                    "canRead": true
                },
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
                "root": {
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
                            "details": true,
                            "name": true,
                            "summary": true
                        }
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "details": true,
                    "name": true,
                    "summary": true
                },
                "schemaLanguage": true,
                "schemaText": true,
                "versionNotes": true
            },
            "codeVersion": {
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
                "root": {
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
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "jsonVariable": true,
                    "name": true
                }
            },
            "inputs": {
                "id": true,
                "index": true,
                "isRequired": true,
                "name": true,
                "standardVersion": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "codeLanguage": true,
                    "default": true,
                    "isComplete": true,
                    "isFile": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "props": true,
                    "variant": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "yup": true,
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
                    "root": {
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
                        }
                    },
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "jsonVariable": true,
                        "name": true
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "helpText": true
                }
            },
            "nodes": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "columnIndex": true,
                "nodeType": true,
                "rowIndex": true,
                "end": {
                    "id": true,
                    "wasSuccessful": true,
                    "suggestedNextRoutineVersions": {
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
                    }
                },
                "routineList": {
                    "id": true,
                    "isOrdered": true,
                    "isOptional": true,
                    "items": {
                        "id": true,
                        "index": true,
                        "isOptional": true,
                        "translations": {
                            "id": true,
                            "language": true,
                            "description": true,
                            "name": true
                        },
                        "routineVersion": {
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
                        }
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "name": true
                }
            },
            "nodeLinks": {
                "id": true,
                "from": {
                    "id": true
                },
                "operation": true,
                "to": {
                    "id": true
                },
                "whens": {
                    "id": true,
                    "condition": true,
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "name": true
                    }
                }
            },
            "outputs": {
                "id": true,
                "index": true,
                "name": true,
                "standardVersion": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "codeLanguage": true,
                    "default": true,
                    "isComplete": true,
                    "isFile": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "props": true,
                    "variant": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "yup": true,
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
                    "root": {
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
                        }
                    },
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "jsonVariable": true,
                        "name": true
                    }
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "helpText": true
                }
            },
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
            "suggestedNextByRoutineVersion": {
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
                    "created_at": true,
                    "updated_at": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        },
        "user": {
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
            "canDelete": true,
            "canUpdate": true,
            "canRead": true
        },
        "lastStep": true,
        "inputs": {
            "id": true,
            "data": true,
            "input": {
                "id": true,
                "index": true,
                "isRequired": true,
                "name": true,
                "standardVersion": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "codeLanguage": true,
                    "default": true,
                    "isComplete": true,
                    "isFile": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "props": true,
                    "variant": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "yup": true,
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
                    "root": {
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
            }
        },
        "outputs": {
            "id": true,
            "data": true,
            "output": {
                "id": true,
                "index": true,
                "name": true,
                "standardVersion": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "codeLanguage": true,
                    "default": true,
                    "isComplete": true,
                    "isFile": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "props": true,
                    "variant": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "yup": true,
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
                    "root": {
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
            }
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
            "step": true,
            "subroutine": {
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
            }
        }
    },
    "__cacheKey": "-2085278056"
};