export const pullRequest_findOne = {
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
} as const;
