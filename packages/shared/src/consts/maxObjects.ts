/* c8 ignore start */
export type ObjectLimitPremium = {
    noPremium: number,
    premium: number,
}

export type ObjectLimitPrivacy = {
    public: number | ObjectLimitPremium,
    private: number | ObjectLimitPremium,
}

export type ObjectLimitOwner = {
    User: number | ObjectLimitPremium | ObjectLimitPrivacy,
    Team: number | ObjectLimitPremium | ObjectLimitPrivacy,
}

export type ObjectLimit = number | ObjectLimitPremium | ObjectLimitPrivacy | ObjectLimitOwner;

/**
 * Contains the maximum number of objects a user/team can have. 
 * Limits vary depending on is the objects are public/private, and whether the 
 * user/team has a premium acccount. 
 * 
 * This is in the shared package because it can be used to display benefits 
 * to switching to premium in the UI, and to validate crud operations in the server
 */
export const MaxObjects = {
    ApiKey: {
        noPremium: 1,
        premium: 5,
    },
    Award: 1000,
    Bookmark: {
        User: {
            private: {
                noPremium: 100,
                premium: 10000,
            },
            public: 0,
        },
        Team: 0,
    },
    BookmarkList: {
        User: {
            private: {
                noPremium: 3,
                premium: 50,
            },
            public: 0,
        },
        Team: 0,
    },
    Chat: {
        User: {
            private: {
                noPremium: 20,
                premium: 500,
            },
            public: {
                noPremium: 20,
                premium: 500,
            },
        },
        Team: {
            private: {
                noPremium: 10,
                premium: 500,
            },
            public: {
                noPremium: 10,
                premium: 500,
            },
        },
    },
    ChatMessage: {
        User: {
            private: {
                noPremium: 5000,
                premium: 50000,
            },
            public: 0,
        },
        Team: 0,
    },
    ChatInvite: {
        User: 0,
        Team: {
            private: {
                noPremium: 10,
                premium: 500,
            },
            public: {
                noPremium: 10,
                premium: 500,
            },
        },
    },
    ChatParticipant: {
        User: 0,
        Team: {
            private: {
                noPremium: 10,
                premium: 500,
            },
            public: {
                noPremium: 10,
                premium: 500,
            },
        },
    },
    Comment: {
        User: {
            private: 0,
            public: 10000,
        },
        Team: 0,
    },
    Email: {
        User: {
            private: 5,
            public: 1,
        },
        Team: {
            private: 0,
            public: 1,
        },
    },
    Issue: {
        User: {
            private: 0,
            public: 10000,
        },
        Team: 0,
    },
    Meeting: {
        User: 0,
        Team: {
            private: 100,
            public: 100,
        },
    },
    MeetingInvite: {
        User: 0,
        Team: 5000,
    },
    Member: {
        User: 0,
        Team: {
            private: {
                noPremium: 5,
                premium: 500,
            },
            public: {
                noPremium: 10,
                premium: 1000,
            },
        },
    },
    MemberInvite: {
        User: 0,
        Team: 1000,
    },
    Notification: 100000,
    NotificationSubscription: {
        User: 1000,
        Team: 0,
    },
    Payment: 1000000,
    Premium: 1000000,
    Phone: {
        noPremium: 1,
        premium: 5,
    },
    PullRequest: 1000000,
    PushDevice: {
        User: 5,
        Team: 0,
    },
    ReactionSummary: 0,
    Report: {
        User: {
            private: 0,
            public: 10000,
        },
        Team: 0,
    },
    ReportResponse: 100000,
    ReminderItem: {
        User: 100000,
        Team: 0,
    },
    Reminder: {
        User: {
            noPremium: 200,
            premium: 10000,
        },
        Team: 0,
    },
    ReminderList: {
        User: {
            noPremium: 25,
            premium: 250,
        },
        Team: 0,
    },
    Resource: {
        private: {
            noPremium: 25,
            premium: 250,
        },
        public: {
            noPremium: 100,
            premium: 2000,
        },
    },
    ResourceVersion: 100000,
    ResourceVersionRelation: 100000,
    RunProject: {
        User: 5000,
        Team: 50000,
    },
    Run: {
        User: 5000,
        Team: 50000,
    },
    RunIO: 100000,
    RunStep: 100000,
    Schedule: {
        User: {
            noPremium: 25,
            premium: 1000,
        },
        Team: {
            noPremium: 25,
            premium: 1000,
        },
    },
    ScheduleException: 100000,
    ScheduleRecurrence: 100000,
    Session: 1000,
    StatsResource: 0,
    StatsSite: 10000000,
    StatsTeam: 0,
    StatsUser: 0,
    Tag: {
        User: 5000,
        Team: 0,
    },
    Team: {
        User: {
            private: {
                noPremium: 1,
                premium: 10,
            },
            public: {
                noPremium: 3,
                premium: 25,
            },
        },
        Team: 0,
    },
    Transfer: {
        User: 5000,
        Team: 5000,
    },
    User: {
        User: 1,
        Team: 0,
    },
    View: 10000000,
    Wallet: {
        User: {
            private: 5,
            public: 0,
        },
        Team: {
            private: {
                noPremium: 1,
                premium: 5,
            },
            public: 0,
        },
    },
} as const;
