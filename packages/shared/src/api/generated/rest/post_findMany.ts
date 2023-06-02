export const post_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "resourceList": {
        "id": true,
        "created_at": true,
        "translations": {
          "id": true,
          "language": true,
          "description": true,
          "name": true
        },
        "resources": {
          "id": true,
          "index": true,
          "link": true,
          "usedFor": true,
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
          }
        }
      },
      "translations": {
        "id": true,
        "language": true,
        "description": true,
        "name": true
      },
      "id": true,
      "created_at": true,
      "updated_at": true,
      "commentsCount": true,
      "repostsCount": true,
      "score": true,
      "bookmarks": true,
      "views": true
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Post"
};
