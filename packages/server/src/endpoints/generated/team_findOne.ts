export const team_findOne = {
    "id": true,
    "publicId": true,
    "bannerImage": true,
    "handle": true,
    "createdAt": true,
    "updatedAt": true,
    "isOpenToNewMembers": true,
    "isPrivate": true,
    "commentsCount": true,
    "membersCount": true,
    "profileImage": true,
    "reportsCount": true,
    "bookmarks": true,
    "tags": {
        "id": true,
        "createdAt": true,
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
    "translations": {
        "id": true,
        "language": true,
        "bio": true,
        "name": true
    },
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
            "createdAt": true,
            "updatedAt": true,
            "isAdmin": true,
            "permissions": true
        }
    },
    "config": true,
    "members": {
        "id": true,
        "publicId": true,
        "createdAt": true,
        "updatedAt": true,
        "isAdmin": true,
        "permissions": true,
        "you": {
            "canDelete": true,
            "canUpdate": true
        },
        "user": {
            "id": true,
            "publicId": true,
            "createdAt": true,
            "updatedAt": true,
            "bannerImage": true,
            "handle": true,
            "isBot": true,
            "isBotDepictingPerson": true,
            "name": true,
            "profileImage": true,
            "bookmarks": true,
            "reportsReceivedCount": true,
            "you": {
                "canDelete": true,
                "canReport": true,
                "canUpdate": true,
                "isBookmarked": true,
                "isViewed": true
            },
            "translations": {
                "id": true,
                "language": true,
                "bio": true
            }
        }
    },
    "__cacheKey": "1337885776"
};