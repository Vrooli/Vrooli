export const comment_findMany = {
    "endCursor": true,
    "threads": {
        "childThreads": {
            "childThreads": {
                "comment": {
                    "id": true,
                    "created_at": true,
                    "updated_at": true,
                    "owner": {
                        "__union": {
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
                "endCursor": true,
                "totalInThread": true
            },
            "comment": {
                "id": true,
                "created_at": true,
                "updated_at": true,
                "owner": {
                    "__union": {
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
            "endCursor": true,
            "totalInThread": true
        },
        "comment": {
            "id": true,
            "created_at": true,
            "updated_at": true,
            "owner": {
                "__union": {
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
        "endCursor": true,
        "totalInThread": true
    },
    "totalThreads": true
};