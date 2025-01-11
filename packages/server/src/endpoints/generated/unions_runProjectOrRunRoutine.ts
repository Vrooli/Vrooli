export const unions_runProjectOrRunRoutine = {
    "edges": {
        "cursor": true,
        "node": {
            "RunProject": {
                "id": true,
                "isPrivate": true,
                "completedComplexity": true,
                "contextSwitches": true,
                "startedAt": true,
                "timeElapsed": true,
                "completedAt": true,
                "name": true,
                "projectVersion": {
                    "id": true,
                    "complexity": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "versionIndex": true,
                    "versionLabel": true,
                    "root": {
                        "id": true,
                        "isPrivate": true
                    },
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "name": true
                    }
                },
                "status": true,
                "stepsCount": true,
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
                    "canUpdate": true,
                    "canRead": true
                },
                "lastStep": true
            },
            "RunRoutine": {
                "id": true,
                "isPrivate": true,
                "completedComplexity": true,
                "contextSwitches": true,
                "startedAt": true,
                "timeElapsed": true,
                "completedAt": true,
                "name": true,
                "status": true,
                "inputsCount": true,
                "outputsCount": true,
                "stepsCount": true,
                "wasRunAutomatically": true,
                "routineVersion": {
                    "id": true,
                    "complexity": true,
                    "isAutomatable": true,
                    "isComplete": true,
                    "isDeleted": true,
                    "isLatest": true,
                    "isPrivate": true,
                    "root": {
                        "id": true,
                        "isInternal": true,
                        "isPrivate": true
                    },
                    "routineType": true,
                    "translations": {
                        "id": true,
                        "language": true,
                        "description": true,
                        "instructions": true,
                        "name": true
                    },
                    "versionIndex": true,
                    "versionLabel": true
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
                    "canUpdate": true,
                    "canRead": true
                },
                "lastStep": true
            }
        }
    },
    "pageInfo": {
        "hasNextPage": true,
        "endCursorRunProject": true,
        "endCursorRunRoutine": true
    },
    "__cacheKey": "-339778715"
};