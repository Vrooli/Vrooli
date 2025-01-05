export const routine_findOne = {
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
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "instructions": true,
            "name": true
        }
    }
};