export const questionAnswer_update = {
  "comments": {
    "translations": {
      "id": true,
      "language": true,
      "text": true
    },
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
    "__typename": "Comment"
  },
  "question": {
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
      "profileImage": true,
      "__typename": "User"
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
        "isPrivate": true,
        "__typename": "Api"
      },
      "Code": {
        "id": true,
        "isPrivate": true,
        "__typename": "Code"
      },
      "Note": {
        "id": true,
        "isPrivate": true,
        "__typename": "Note"
      },
      "Project": {
        "id": true,
        "isPrivate": true,
        "__typename": "Project"
      },
      "Routine": {
        "id": true,
        "isInternal": true,
        "isPrivate": true,
        "__typename": "Routine"
      },
      "Standard": {
        "id": true,
        "isPrivate": true,
        "__typename": "Standard"
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
        },
        "__typename": "Team"
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
      },
      "__typename": "Tag"
    },
    "you": {
      "reaction": true
    },
    "__typename": "Question"
  },
  "translations": {
    "id": true,
    "language": true,
    "text": true
  },
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
    "profileImage": true,
    "__typename": "User"
  },
  "score": true,
  "bookmarks": true,
  "isAccepted": true,
  "commentsCount": true,
  "__typename": "QuestionAnswer"
} as const;
