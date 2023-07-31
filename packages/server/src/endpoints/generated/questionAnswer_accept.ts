export const questionAnswer_accept = {
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
      "User": {
        "id": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "name": true,
        "profileImage": true,
        "__typename": "User"
      },
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
        },
        "__typename": "Organization"
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
      "bannerImage": true,
      "handle": true,
      "isBot": true,
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
      "Note": {
        "id": true,
        "isPrivate": true,
        "__typename": "Note"
      },
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
        },
        "__typename": "Organization"
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
      "SmartContract": {
        "id": true,
        "isPrivate": true,
        "__typename": "SmartContract"
      },
      "Standard": {
        "id": true,
        "isPrivate": true,
        "__typename": "Standard"
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
    "bannerImage": true,
    "handle": true,
    "isBot": true,
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
