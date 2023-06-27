
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
    Organization: number | ObjectLimitPremium | ObjectLimitPrivacy,
}

export type ObjectLimit = number | ObjectLimitPremium | ObjectLimitPrivacy | ObjectLimitOwner;

/**
 * Contains the maximum number of objects a user/organization can have. 
 * Limits vary depending on is the objects are public/private, and whether the 
 * user/organization has a premium acccount. 
 * 
 * This is in the shared package because it can be used to display benefits 
 * to switching to premium in the UI, and to validate crud operations in the server
 */
export const MaxObjects = {
    Api: {
        User: {
            private: {
                noPremium: 1,
                premium: 10,
            },
            public: {
                noPremium: 3,
                premium: 100,
            },
        },
        Organization: {
            private: {
                noPremium: 3,
                premium: 25,
            },
            public: {
                noPremium: 5,
                premium: 100,
            },
        },
    },
    ApiKey: {
        noPremium: 1,
        premium: 5,
    },
    ApiVersion: 100000,
    Award: 1000,
    Bookmark: {
        User: {
            private: {
                noPremium: 100,
                premium: 10000,
            },
            public: 0,
        },
        Organization: 0,
    },
    BookmarkList: {
        User: {
            private: {
                noPremium: 3,
                premium: 50,
            },
            public: 0,
        },
        Organization: 0,
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
        Organization: {
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
        Organization: 0,
    },
    ChatInvite: {
        User: 0,
        Organization: {
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
        Organization: {
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
        Organization: 0,
    },
    Email: {
        User: {
            private: 5,
            public: 1,
        },
        Organization: {
            private: 0,
            public: 1,
        },
    },
    FocusMode: {
        User: {
            noPremium: 2,
            premium: 15,
        },
        Organization: 0,
    },
    FocusModeFilter: {
        User: {
            private: {
                noPremium: 25,
                premium: 100,
            },
            public: 0,
        },
        Organization: 0,
    },
    Issue: {
        User: {
            private: 0,
            public: 10000,
        },
        Organization: 0,
    },
    Label: {
        User: {
            private: {
                noPremium: 20,
                premium: 100,
            },
            public: {
                noPremium: 20,
                premium: 100,
            },
        },
        Organization: {
            private: {
                noPremium: 20,
                premium: 100,
            },
            public: {
                noPremium: 20,
                premium: 100,
            },
        },
    },
    Meeting: {
        User: 0,
        Organization: {
            private: 100,
            public: 100,
        },
    },
    MeetingInvite: {
        User: 0,
        Organization: 5000,
    },
    Member: {
        User: 0,
        Organization: {
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
        Organization: 1000,
    },
    Node: 100000,
    NodeEnd: 100000,
    NodeLink: 100000,
    NodeLinkWhen: 100000,
    NodeLoop: 100000,
    NodeLoopWhile: 100000,
    NodeRoutineList: 100000,
    NodeRoutineListItem: 100000,
    Note: {
        User: {
            private: {
                noPremium: 25,
                premium: 1000,
            },
            public: {
                noPremium: 50,
                premium: 2000,
            },
        },
        Organization: {
            private: {
                noPremium: 25,
                premium: 1000,
            },
            public: {
                noPremium: 50,
                premium: 2000,
            },
        },
    },
    NoteVersion: 100000,
    Notification: 100000,
    NotificationSubscription: {
        User: 1000,
        Organization: 0,
    },
    Organization: {
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
        Organization: 0,
    },
    Payment: 1000000,
    Premium: 1000000,
    Phone: {
        noPremium: 1,
        premium: 5,
    },
    Post: {
        noPremium: 10000,
        premium: 100000,
    },
    Project: {
        private: {
            noPremium: 3,
            premium: 100,
        },
        public: {
            noPremium: 25,
            premium: 250,
        },
    },
    ProjectVersion: 100000,
    ProjectVersionDirectory: 1000000,
    PushDevice: {
        User: 5,
        Organization: 0,
    },
    Question: {
        User: 5000,
        Organization: 0,
    },
    QuestionAnswer: {
        User: 10000,
        Organization: 0,
    },
    Quiz: {
        User: 2000,
        Organization: 0,
    },
    QuizQuestion: {
        User: 200000,
        Organization: 0,
    },
    QuizQuestionResponse: {
        User: 200000,
        Organization: 0,
    },
    Report: {
        User: {
            private: 0,
            public: 10000,
        },
        Organization: 0,
    },
    ReminderItem: {
        User: 100000,
        Organization: 0,
    },
    Reminder: {
        User: {
            noPremium: 200,
            premium: 10000,
        },
        Organization: 0,
    },
    ReminderList: {
        User: {
            noPremium: 25,
            premium: 250,
        },
        Organization: 0,
    },
    Resource: 50000,
    ResourceList: 50000,
    Role: {
        User: 0,
        Organization: {
            private: {
                noPremium: 5,
                premium: 50,
            },
            public: {
                noPremium: 5,
                premium: 50,
            },
        },
    },
    Routine: {
        private: {
            noPremium: 25,
            premium: 250,
        },
        public: {
            noPremium: 100,
            premium: 2000,
        },
    },
    RoutineVersion: 100000,
    RunProject: {
        User: 5000,
        Organization: 50000,
    },
    RunRoutine: {
        User: 5000,
        Organization: 50000,
    },
    RunRoutineInput: 100000,
    RunRoutineStep: 100000,
    Schedule: {
        User: {
            noPremium: 25,
            premium: 1000,
        },
        Organization: {
            noPremium: 25,
            premium: 1000,
        },
    },
    ScheduleException: 100000,
    ScheduleRecurrence: 100000,
    SmartContract: {
        private: {
            noPremium: 6,
            premium: 50,
        },
        public: {
            noPremium: 10,
            premium: 200,
        },
    },
    SmartContractVersion: 100000,
    Standard: {
        private: {
            noPremium: 5,
            premium: 100,
        },
        public: {
            noPremium: 100,
            premium: 1000,
        },
    },
    StandardVersion: 100000,
    Tag: {
        User: 5000,
        Organization: 0,
    },
    Transfer: {
        User: 5000,
        Organization: 5000,
    },
    User: 0,
    Wallet: {
        User: {
            private: 5,
            public: 0,
        },
        Organization: {
            private: {
                noPremium: 1,
                premium: 5,
            },
            public: 0,
        },
    },
} as const;
