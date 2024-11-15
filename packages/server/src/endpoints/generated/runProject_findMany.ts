export const runProject_findMany = {
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
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RunProject"
} as const;
