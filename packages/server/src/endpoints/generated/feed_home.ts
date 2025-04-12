export const feed_home = {
    "recommended": {
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
    },
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
        },
        "reminderList": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "focusMode": {
                "id": true,
                "name": true,
                "description": true,
                "you": {
                    "canDelete": true,
                    "canRead": true,
                    "canUpdate": true
                },
                "labels": {
                    "id": true,
                    "color": true,
                    "label": true
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
                "schedule": {
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
                    }
                }
            }
        }
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
    },
    "schedules": {
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
            }
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
                    "isPrivate": true
                },
                "translations": {
                    "id": true,
                    "language": true,
                    "description": true,
                    "name": true
                }
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
            "lastStep": true
        },
        "runRoutines": {
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
            "lastStep": true
        }
    },
    "__cacheKey": "-103616588"
};