export const scheduleRecurrence_create = {
  "id": true,
  "recurrenceType": true,
  "interval": true,
  "dayOfWeek": true,
  "dayOfMonth": true,
  "month": true,
  "endDate": true,
  "schedule": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "startTime": true,
    "endTime": true,
    "timezone": true,
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
          },
          "__typename": "Team"
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
          "profileImage": true,
          "__typename": "User"
        }
      },
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
      "standardsCount": true,
      "__typename": "Label"
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
          },
          "__typename": "Tag"
        },
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
            "label": true,
            "__typename": "Label"
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
                "isComplete": true,
                "__typename": "ReminderItem"
              },
              "__typename": "Reminder"
            },
            "__typename": "ReminderList"
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
              "newEndTime": true,
              "__typename": "ScheduleException"
            },
            "recurrences": {
              "id": true,
              "recurrenceType": true,
              "interval": true,
              "dayOfWeek": true,
              "dayOfMonth": true,
              "month": true,
              "endDate": true,
              "__typename": "ScheduleRecurrence"
            },
            "__typename": "Schedule"
          },
          "__typename": "FocusMode"
        },
        "__typename": "FocusModeFilter"
      },
      "labels": {
        "id": true,
        "color": true,
        "label": true,
        "__typename": "Label"
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
            "isComplete": true,
            "__typename": "ReminderItem"
          },
          "__typename": "Reminder"
        },
        "__typename": "ReminderList"
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
      "__typename": "FocusMode"
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
        },
        "__typename": "Team"
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
          },
          "__typename": "Team"
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
              },
              "__typename": "Team"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true
            },
            "__typename": "Role"
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
            },
            "__typename": "User"
          },
          "__typename": "Member"
        },
        "__typename": "Role"
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
        "profileImage": true,
        "__typename": "User"
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
        },
        "__typename": "MeetingInvite"
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
            },
            "__typename": "Team"
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
            "profileImage": true,
            "__typename": "User"
          }
        },
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
        "standardsCount": true,
        "__typename": "Label"
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "link": true,
        "name": true
      },
      "__typename": "Meeting"
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
        },
        "__typename": "Team"
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
        "__typename": "User"
      },
      "you": {
        "canDelete": true,
        "canUpdate": true,
        "canRead": true
      },
      "lastStep": true,
      "projectVersion": {
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
                },
                "__typename": "Team"
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
                "profileImage": true,
                "__typename": "User"
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            },
            "__typename": "Label"
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
              },
              "__typename": "Team"
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
              "profileImage": true,
              "__typename": "User"
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
            },
            "__typename": "Tag"
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
          "__typename": "Project"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "__typename": "ProjectVersion"
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
              "isPrivate": true,
              "__typename": "Project"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "name": true
            },
            "__typename": "ProjectVersion"
          },
          "__typename": "ProjectVersionDirectory"
        },
        "__typename": "RunProjectStep"
      },
      "__typename": "RunProject"
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
        },
        "__typename": "Team"
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
        "__typename": "User"
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
            "isComplete": true,
            "isFile": true,
            "isLatest": true,
            "isPrivate": true,
            "default": true,
            "standardType": true,
            "props": true,
            "yup": true,
            "versionIndex": true,
            "versionLabel": true,
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
                    },
                    "__typename": "Team"
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
                    "profileImage": true,
                    "__typename": "User"
                  }
                },
                "you": {
                  "canDelete": true,
                  "canUpdate": true
                },
                "__typename": "Label"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
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
                },
                "__typename": "Tag"
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
              "__typename": "Standard"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "jsonVariable": true,
              "name": true
            },
            "__typename": "StandardVersion"
          },
          "__typename": "RoutineVersionInput"
        },
        "__typename": "RunRoutineInput"
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
            "isComplete": true,
            "isFile": true,
            "isLatest": true,
            "isPrivate": true,
            "default": true,
            "standardType": true,
            "props": true,
            "yup": true,
            "versionIndex": true,
            "versionLabel": true,
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
                    },
                    "__typename": "Team"
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
                    "profileImage": true,
                    "__typename": "User"
                  }
                },
                "you": {
                  "canDelete": true,
                  "canUpdate": true
                },
                "__typename": "Label"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
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
                },
                "__typename": "Tag"
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
              "__typename": "Standard"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "jsonVariable": true,
              "name": true
            },
            "__typename": "StandardVersion"
          },
          "__typename": "RoutineVersionOutput"
        },
        "__typename": "RunRoutineOutput"
      },
      "routineVersion": {
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
              "profileImage": true,
              "__typename": "User"
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
            },
            "__typename": "PullRequest"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
                }
              },
              "you": {
                "canDelete": true,
                "canUpdate": true
              },
              "__typename": "Label"
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
                },
                "__typename": "Team"
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
                "profileImage": true,
                "__typename": "User"
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
              },
              "__typename": "Tag"
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
                "isPrivate": true,
                "__typename": "Api"
              },
              "translations": {
                "id": true,
                "language": true,
                "details": true,
                "name": true,
                "summary": true
              },
              "__typename": "ApiVersion"
            },
            "__typename": "Api"
          },
          "translations": {
            "id": true,
            "language": true,
            "details": true,
            "name": true,
            "summary": true
          },
          "schemaText": true,
          "versionNotes": true,
          "__typename": "ApiVersion"
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
              "profileImage": true,
              "__typename": "User"
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
            },
            "__typename": "PullRequest"
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
              },
              "__typename": "Resource"
            },
            "__typename": "ResourceList"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
                }
              },
              "you": {
                "canDelete": true,
                "canUpdate": true
              },
              "__typename": "Label"
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
                },
                "__typename": "Team"
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
                "profileImage": true,
                "__typename": "User"
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
              },
              "__typename": "Tag"
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
                "isPrivate": true,
                "__typename": "Code"
              },
              "translations": {
                "id": true,
                "language": true,
                "description": true,
                "jsonVariable": true,
                "name": true
              },
              "__typename": "CodeVersion"
            },
            "__typename": "Code"
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "jsonVariable": true,
            "name": true
          },
          "__typename": "CodeVersion"
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
            "isComplete": true,
            "isFile": true,
            "isLatest": true,
            "isPrivate": true,
            "default": true,
            "standardType": true,
            "props": true,
            "yup": true,
            "versionIndex": true,
            "versionLabel": true,
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
                    },
                    "__typename": "Team"
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
                    "profileImage": true,
                    "__typename": "User"
                  }
                },
                "you": {
                  "canDelete": true,
                  "canUpdate": true
                },
                "__typename": "Label"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
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
                },
                "__typename": "Tag"
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
              "__typename": "Standard"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "jsonVariable": true,
              "name": true
            },
            "__typename": "StandardVersion"
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "helpText": true
          },
          "__typename": "RoutineVersionInput"
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
                "isPrivate": true,
                "__typename": "Routine"
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
              "__typename": "RoutineVersion"
            },
            "__typename": "NodeEnd"
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
                  "isPrivate": true,
                  "__typename": "Routine"
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
                "__typename": "RoutineVersion"
              },
              "__typename": "NodeRoutineListItem"
            },
            "__typename": "NodeRoutineList"
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          },
          "__typename": "Node"
        },
        "nodeLinks": {
          "id": true,
          "from": {
            "id": true,
            "__typename": "Node"
          },
          "operation": true,
          "to": {
            "id": true,
            "__typename": "Node"
          },
          "whens": {
            "id": true,
            "condition": true,
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "name": true
            },
            "__typename": "NodeLinkWhen"
          },
          "__typename": "NodeLink"
        },
        "outputs": {
          "id": true,
          "index": true,
          "name": true,
          "standardVersion": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "isComplete": true,
            "isFile": true,
            "isLatest": true,
            "isPrivate": true,
            "default": true,
            "standardType": true,
            "props": true,
            "yup": true,
            "versionIndex": true,
            "versionLabel": true,
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
                    },
                    "__typename": "Team"
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
                    "profileImage": true,
                    "__typename": "User"
                  }
                },
                "you": {
                  "canDelete": true,
                  "canUpdate": true
                },
                "__typename": "Label"
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
                  },
                  "__typename": "Team"
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
                  "profileImage": true,
                  "__typename": "User"
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
                },
                "__typename": "Tag"
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
              "__typename": "Standard"
            },
            "translations": {
              "id": true,
              "language": true,
              "description": true,
              "jsonVariable": true,
              "name": true
            },
            "__typename": "StandardVersion"
          },
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "helpText": true
          },
          "__typename": "RoutineVersionOutput"
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
            "profileImage": true,
            "__typename": "User"
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
          },
          "__typename": "PullRequest"
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
            },
            "__typename": "Resource"
          },
          "__typename": "ResourceList"
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
                },
                "__typename": "Team"
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
                "profileImage": true,
                "__typename": "User"
              }
            },
            "you": {
              "canDelete": true,
              "canUpdate": true
            },
            "__typename": "Label"
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
              },
              "__typename": "Team"
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
              "profileImage": true,
              "__typename": "User"
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
            },
            "__typename": "Tag"
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
              "isPrivate": true,
              "__typename": "Routine"
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
            "__typename": "RoutineVersion"
          },
          "__typename": "Routine"
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
            "isPrivate": true,
            "__typename": "Routine"
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
          "__typename": "RoutineVersion"
        },
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "instructions": true,
          "name": true
        },
        "__typename": "RoutineVersion"
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
            "isPrivate": true,
            "__typename": "Routine"
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
          "__typename": "RoutineVersion"
        },
        "__typename": "RunRoutineStep"
      },
      "__typename": "RunRoutine"
    },
    "__typename": "Schedule"
  },
  "__typename": "ScheduleRecurrence"
} as const;
