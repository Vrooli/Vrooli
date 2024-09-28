export const report_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "details": true,
      "language": true,
      "reason": true,
      "responsesCount": true,
      "status": true,
      "you": {
        "canDelete": true,
        "canRespond": true,
        "canUpdate": true,
        "isOwn": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "Report"
} as const;
