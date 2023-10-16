export const chatMessage_findTree = {
  "hasMoreUp": true,
  "hasMoreDown": true,
  "messages": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "sequence": true,
    "versionIndex": true,
    "parent": {
      "id": true,
      "created_at": true
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
    "score": true,
    "reactionSummaries": {
      "emoji": true,
      "count": true
    },
    "reportsCount": true,
    "you": {
      "canDelete": true,
      "canReply": true,
      "canReport": true,
      "canUpdate": true,
      "canReact": true,
      "reaction": true
    }
  },
  "__typename": "ChatMessage"
} as const;