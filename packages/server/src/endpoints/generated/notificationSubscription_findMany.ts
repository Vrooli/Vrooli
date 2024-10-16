export const notificationSubscription_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "silent": true,
      "object": {
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
        "Comment": {
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
          "translations": {
            "id": true,
            "language": true,
            "text": true
          }
        },
        "Issue": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "closedAt": true,
          "referencedVersionId": true,
          "status": true,
          "to": {
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
          "commentsCount": true,
          "reportsCount": true,
          "score": true,
          "bookmarks": true,
          "views": true,
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
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
            "you": {
              "canDelete": true,
              "canUpdate": true
            }
          },
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
            "you": {
              "canDelete": true,
              "canUpdate": true
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
            },
            "labels": {
              "id": true,
              "created_at": true,
              "updated_at": true,
              "color": true,
              "label": true,
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
            "runProjects": {
              "id": true,
              "isPrivate": true,
              "completedComplexity": true,
              "contextSwitches": true,
              "startedAt": true,
              "timeElapsed": true,
              "completedAt": true,
              "name": true,
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
            "link": true,
            "name": true
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
        "PullRequest": {
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
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          }
        },
        "Quiz": {
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
          "attemptsCount": true,
          "quizQuestionsCount": true,
          "project": {
            "id": true,
            "isPrivate": true
          },
          "routine": {
            "id": true,
            "isInternal": true,
            "isPrivate": true
          },
          "you": {
            "canDelete": true,
            "canBookmark": true,
            "canUpdate": true,
            "canRead": true,
            "canReact": true,
            "hasCompleted": true,
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
        "Report": {
          "id": true,
          "created_at": true,
          "updated_at": true,
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
        },
        "Team": {
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
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "NotificationSubscription"
} as const;
