export const notification_getSettings = {
    "includedEmails": {
        "id": true,
        "emailAddress": true,
        "verified": true
    },
    "includedSms": {
        "id": true,
        "phoneNumber": true,
        "verified": true
    },
    "includedPush": {
        "id": true,
        "expires": true,
        "name": true
    },
    "toEmails": true,
    "toSms": true,
    "toPush": true,
    "dailyLimit": true,
    "enabled": true,
    "categories": {
        "category": true,
        "enabled": true,
        "dailyLimit": true,
        "toEmails": true,
        "toSms": true,
        "toPush": true
    }
};