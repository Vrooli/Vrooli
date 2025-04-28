export const chatInvite_createMany = {
    "id": true,
    "created_at": true,
    "updated_at": true,
    "message": true,
    "status": true,
    "user": {
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
    "you": {
        "canDelete": true,
        "canUpdate": true
    },
    "chat": {
        "id": true,
        "created_at": true,
        "updated_at": true,
        "openToAnyoneWithInvite": true,
        "participants": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "user": {
                "id": true,
                "created_at": true,
                "updated_at": true,
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
        "restrictedToRoles": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "name": true,
            "permissions": true,
            "membersCount": true,
            "team": {
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
            "translations": {
                "id": true,
                "language": true,
                "description": true
            },
            "members": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "isAdmin": true,
                "permissions": true,
                "roles": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "name": true,
                    "permissions": true,
                    "membersCount": true,
                    "team": {
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
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true
                    }
                },
                "you": {
                    "canDelete": true,
                    "canUpdate": true
                },
                "user": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
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
            }
        },
        "team": {
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
        "participantsCount": true,
        "invitesCount": true,
        "you": {
            "canDelete": true,
            "canInvite": true,
            "canUpdate": true
        },
        "translations": {
            "id": true,
            "language": true,
            "description": true,
            "name": true
        }
    },
    "__cacheKey": "752705085"
};