export const tag_findMany = {
  "edges": {
    "cursor": true,
    "node": {
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
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Tag"
};
