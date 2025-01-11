export const questionAnswer_createOne = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "createdBy": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "bannerImage": true,
        "handle": true,
        "isBot": true,
        "isBotDepictingPerson": true,
        "name": true,
        "profileImage": true
    },
    "score": true,
    "bookmarks": true,
    "isAccepted": true,
    "commentsCount": true,
    "comments": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "owner": {
            "Team": {
                "id": true,
                "bannerImage": true,
                "handle": true,
                "profileImage": true,
                "you": {
                    "canAddMembers": true,
                    "canDelete": true,
                    "canBookmark": true,
                    "canReport": true,
                    "canUpdate": true,
                    "canRead": true,
                    "isBookmarked": true,
                    "isViewed": true,
                    "yourMembership": {
                        "id": true,
                        "created_at": true,
                        "updated_at": true,
                        "isAdmin": true,
                        "permissions": true
                    }
                }
            },
            "User": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "bannerImage": true,
                "handle": true,
                "isBot": true,
                "isBotDepictingPerson": true,
                "name": true,
                "profileImage": true
            }
        },
        "score": true,
        "bookmarks": true,
        "reportsCount": true,
        "you": {
            "canDelete": true,
            "canBookmark": true,
            "canReply": true,
            "canReport": true,
            "canUpdate": true,
            "canReact": true,
            "isBookmarked": true,
            "reaction": true
        },
        "translations": {
            "id": true,
            "language": true,
            "text": true
        }
    },
    "question": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "createdBy": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true
        },
        "hasAcceptedAnswer": true,
        "isPrivate": true,
        "score": true,
        "bookmarks": true,
        "answersCount": true,
        "commentsCount": true,
        "reportsCount": true,
        "forObject": {
            "Api": {
                "id": true,
                "isPrivate": true
            },
            "Code": {
                "id": true,
                "isPrivate": true
            },
            "Note": {
                "id": true,
                "isPrivate": true
            },
            "Project": {
                "id": true,
                "isPrivate": true
            },
            "Routine": {
                "id": true,
                "isInternal": true,
                "isPrivate": true
            },
            "Standard": {
                "id": true,
                "isPrivate": true
            },
            "Team": {
                "id": true,
                "bannerImage": true,
                "handle": true,
                "profileImage": true,
                "you": {
                    "canAddMembers": true,
                    "canDelete": true,
                    "canBookmark": true,
                    "canReport": true,
                    "canUpdate": true,
                    "canRead": true,
                    "isBookmarked": true,
                    "isViewed": true,
                    "yourMembership": {
                        "id": true,
                        "created_at": true,
                        "updated_at": true,
                        "isAdmin": true,
                        "permissions": true
                    }
                }
            }
        },
        "tags": {
            "id": true,
            "created_at": true,
            "tag": true,
            "bookmarks": true,
            "translations": {
                "id": true,
                "language": true,
                "description": true
            },
            "you": {
                "isOwn": true,
                "isBookmarked": true
            }
        },
        "you": {
            "reaction": true
        }
    },
    "translations": {
        "id": true,
        "language": true,
        "text": true
    },
    "__cacheKey": "1125374912"
};