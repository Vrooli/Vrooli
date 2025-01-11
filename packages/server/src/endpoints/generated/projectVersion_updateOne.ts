export const projectVersion_updateOne = {
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
    "you": {
        "canComment": true,
        "canCopy": true,
        "canDelete": true,
        "canReport": true,
        "canUpdate": true,
        "canUse": true,
        "canRead": true
    },
    "directories": {
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
        },
        "children": {
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
        },
        "childApiVersions": {
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
            "schemaLanguage": true,
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
                "details": true,
                "name": true,
                "summary": true
            }
        },
        "childCodeVersions": {
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
        "childNoteVersions": {
            "id": true,
            "created_at": true,
            "updated_at": true,
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
                "name": true,
                "pages": {
                    "id": true,
                    "pageIndex": true,
                    "text": true
                }
            }
        },
        "childProjectVersions": {
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
                "name": true
            }
        },
        "childRoutineVersions": {
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
            "nodesCount": true,
            "nodeLinksCount": true,
            "outputsCount": true,
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
                "created_at": true,
                "updated_at": true,
                "isInternal": true,
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
                "instructions": true,
                "name": true
            }
        },
        "childStandardVersions": {
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
        "childTeams": {
            "id": true,
            "bannerImage": true,
            "handle": true,
            "created_at": true,
            "updated_at": true,
            "isOpenToNewMembers": true,
            "isPrivate": true,
            "commentsCount": true,
            "membersCount": true,
            "profileImage": true,
            "reportsCount": true,
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
                    "created_at": true,
                    "updated_at": true,
                    "isAdmin": true,
                    "permissions": true
                }
            }
        },
        "parentDirectory": {
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
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
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
    },
    "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
    },
    "versionNotes": true,
    "__cacheKey": "1424441677"
};