export const runRoutine_findMany = {
  "edges": {
    "cursor": true,
    "node": {
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
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RunRoutine"
} as const;
