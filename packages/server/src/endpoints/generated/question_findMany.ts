export const question_findMany = {
    "edges": {
        "cursor": true,
        "node": {
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
                "__union": {
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
            },
            "translations": {
                "id": true,
                "language": true,
                "description": true,
                "name": true
            }
        }
    },
    "pageInfo": {
        "endCursor": true,
        "hasNextPage": true
    }
};