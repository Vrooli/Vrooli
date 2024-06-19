export const report_create = {
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
  "__typename": "Report"
} as const;
