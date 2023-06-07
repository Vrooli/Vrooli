export const pullRequest_accept = {
  "translations": {
    "id": true,
    "language": true,
    "text": true
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "mergedOrRejectedAt": true,
  "commentsCount": true,
  "status": true,
  "createdBy": {
    "id": true,
    "isBot": true,
    "name": true,
    "handle": true,
    "__typename": "User"
  },
  "you": {
    "canComment": true,
    "canDelete": true,
    "canReport": true,
    "canUpdate": true
  },
  "__typename": "PullRequest"
} as const;