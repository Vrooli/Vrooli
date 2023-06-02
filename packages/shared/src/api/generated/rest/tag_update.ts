export const tag_update = {
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
} as const;
