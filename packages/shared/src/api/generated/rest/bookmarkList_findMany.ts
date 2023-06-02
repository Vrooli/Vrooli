export const bookmarkList_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "label": true,
      "bookmarksCount": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "BookmarkList"
} as const;
