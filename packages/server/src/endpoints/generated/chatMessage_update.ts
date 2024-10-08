export const chatMessage_update = {
  "id": true,
  "created_at": true,
  "updated_at": true,
  "sequence": true,
  "versionIndex": true,
  "parent": {
    "id": true,
    "created_at": true,
    "__typename": "ChatMessage"
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
  "score": true,
  "reactionSummaries": {
    "emoji": true,
    "count": true,
    "__typename": "ReactionSummary"
  },
  "reportsCount": true,
  "you": {
    "canDelete": true,
    "canReply": true,
    "canReport": true,
    "canUpdate": true,
    "canReact": true,
    "reaction": true
  },
  "translations": {
    "id": true,
    "language": true,
    "text": true
  },
  "__typename": "ChatMessage"
} as const;
