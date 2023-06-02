export const reportResponse_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "created_at": true,
      "updated_at": true,
      "actionSuggested": true,
      "details": true,
      "language": true,
      "you": {
        "canDelete": true,
        "canUpdate": true
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "ReportResponse"
};
