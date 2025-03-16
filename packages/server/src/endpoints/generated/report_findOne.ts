export const report_findOne = {
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
        }
    },
    "__cacheKey": "-1921025353"
};