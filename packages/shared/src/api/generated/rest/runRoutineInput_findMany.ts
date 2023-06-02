export const runRoutineInput_findMany = {
  "edges": {
    "cursor": true,
    "node": {
      "id": true,
      "data": true,
      "input": {
        "id": true,
        "index": true,
        "isRequired": true,
        "name": true,
        "standardVersion": {
          "translations": {
            "id": true,
            "language": true,
            "description": true,
            "jsonVariable": true,
            "name": true
          },
          "id": true,
          "created_at": true,
          "updated_at": true,
          "isComplete": true,
          "isFile": true,
          "isLatest": true,
          "isPrivate": true,
          "default": true,
          "standardType": true,
          "props": true,
          "yup": true,
          "versionIndex": true,
          "versionLabel": true,
          "commentsCount": true,
          "directoryListingsCount": true,
          "forksCount": true,
          "reportsCount": true,
          "you": {
            "canComment": true,
            "canCopy": true,
            "canDelete": true,
            "canReport": true,
            "canUpdate": true,
            "canUse": true,
            "canRead": true
          }
        }
      }
    }
  },
  "pageInfo": {
    "endCursor": true,
    "hasNextPage": true
  },
  "__typename": "RunRoutineInput"
};
