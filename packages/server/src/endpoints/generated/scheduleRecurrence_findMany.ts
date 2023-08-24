export const scheduleRecurrence_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "schedule": {
        "labels": {
          "id": true,
          "created_at": true,
          "updated_at": true,
          "color": true,
          "label": true,
          "owner": {
            "Organization": {
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
          "labels": {
            "id": true,
            "color": true,
            "label": true
          },
          "id": true,
          "name": true,
          "description": true
        },
        "meetings": {
          "labels": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "color": true,
            "label": true,
            "owner": {
              "Organization": {
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
                "name": true,
                "profileImage": true
              }
            },
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
          },
          "id": true,
          "openToAnyoneWithInvite": true,
          "showOnOrganizationProfile": true,
          "organization": {
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
                "organization": {
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
              }
            },
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "permissions": true,
            "membersCount": true,
            "organization": {
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
          "attendeesCount": true,
          "invitesCount": true,
          "you": {
            "canDelete": true,
            "canInvite": true,
            "canUpdate": true
          }
        },
        "runProjects": {
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
          "organization": {
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
            "name": true,
            "profileImage": true
          },
          "you": {
            "canDelete": true,
            "canUpdate": true,
            "canRead": true
          }
        },
        "runRoutines": {
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
          "inputsCount": true,
          "wasRunAutomatically": true,
          "organization": {
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
            "name": true,
            "profileImage": true
          },
          "you": {
            "canDelete": true,
            "canUpdate": true,
            "canRead": true
          }
        },
        "id": true,
        "created_at": true,
        "updated_at": true,
        "startTime": true,
        "endTime": true,
        "timezone": true
      },
      "id": true,
      "recurrenceType": true,
      "interval": true,
      "dayOfWeek": true,
      "dayOfMonth": true,
      "month": true,
      "endDate": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ScheduleRecurrence"
} as const;
