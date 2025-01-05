export const reminder_updateOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "name": true,
    "description": true,
    "dueDate": true,
    "index": true,
    "isComplete": true,
    "reminderItems": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "name": true,
        "description": true,
        "dueDate": true,
        "index": true,
        "isComplete": true
    },
    "reminderList": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "focusMode": {
            "id": true,
            "name": true,
            "description": true,
            "you": {
                "canDelete": true,
                "canRead": true,
                "canUpdate": true
            },
            "labels": {
                "id": true,
                "color": true,
                "label": true
            },
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
            "schedule": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "startTime": true,
                "endTime": true,
                "timezone": true,
                "exceptions": {
                    "id": true,
                    "originalStartTime": true,
                    "newStartTime": true,
                    "newEndTime": true
                },
                "recurrences": {
                    "id": true,
                    "recurrenceType": true,
                    "interval": true,
                    "dayOfWeek": true,
                    "dayOfMonth": true,
                    "month": true,
                    "endDate": true
                }
            }
        }
    }
};