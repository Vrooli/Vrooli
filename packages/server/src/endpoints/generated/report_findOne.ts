export const report_findOne = {
  "responses": {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "actionSuggested": true,
    "details": true,
    "language": true,
    "you": {
      "canDelete": true,
      "canUpdate": true
    },
    "__typename": "ReportResponse"
  },
  "id": true,
  "created_at": true,
  "updated_at": true,
  "details": true,
  "language": true,
  "reason": true,
  "responsesCount": true,
  "you": {
    "canDelete": true,
    "canRespond": true,
    "canUpdate": true
  },
  "__typename": "Report"
} as const;
