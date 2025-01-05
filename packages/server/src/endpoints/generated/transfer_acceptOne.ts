export const transfer_acceptOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "mergedOrRejectedAt": true,
    "status": true,
    "fromOwner": {
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
    "toOwner": {
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
    "object": {
        "__union": {
            "Api": {
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
                    "__union": {
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
                "versions": {
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
                    "translations": {
                        "id": true,
                        "language": true,
                        "details": true,
                        "name": true,
                        "summary": true
                    }
                }
            },
            "Code": {
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
                    "__union": {
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
                "versions": {
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
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "jsonVariable": true,
                        "name": true
                    }
                }
            },
            "Note": {
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
                    "__union": {
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
                "versions": {
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
                }
            },
            "Project": {
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
                    "__union": {
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
                    "__union": {
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
                    "nodesCount": true,
                    "nodeLinksCount": true,
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
            },
            "Standard": {
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
                    "__union": {
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
                "versions": {
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
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "jsonVariable": true,
                        "name": true
                    }
                }
            }
        }
    },
    "you": {
        "canDelete": true,
        "canUpdate": true
    }
};