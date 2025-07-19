import type { EmailLogInInput, EmailRequestPasswordChangeInput, EmailResetPasswordInput, ValidateSessionInput, SwitchCurrentAccountInput } from "../../../api/types.js";
import { type ModelTestFixtures, TypedTestDataFactory, createTypedFixtures } from "../../../validation/models/__test/validationTestUtils.js";

// Valid Snowflake IDs for testing
const validIds = {
    id1: "100000000000000001",
    id2: "100000000000000002",
    id3: "100000000000000003",
    id4: "100000000000000004",
};

// Email login test fixtures
export const emailLogInFixtures: ModelTestFixtures<EmailLogInInput, never> = {
    minimal: {
        create: {
            email: "test@example.com",
            password: "SecurePassword123!",
        },
        update: null as never,
    },
    complete: {
        create: {
            email: "complete@example.com",
            password: "CompletePassword123!",
            verificationCode: "123456",
        },
        update: null as never,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing email and password
            } as EmailLogInInput,
            update: null as never,
        },
        invalidTypes: {
            create: {
                email: 123, // Should be string
                password: true, // Should be string
                verificationCode: 456, // Should be string
            } as unknown as EmailLogInInput,
            update: null as never,
        },
        invalidEmail: {
            create: {
                email: "invalid-email",
                password: "ValidPassword123!",
            },
            update: null as never,
        },
    },
    edgeCases: {
        onlyEmail: {
            create: {
                email: "only@example.com",
            },
            update: null as never,
        },
        onlyPassword: {
            create: {
                password: "OnlyPassword123!",
            },
            update: null as never,
        },
        onlyVerificationCode: {
            create: {
                verificationCode: "654321",
            },
            update: null as never,
        },
    },
};

// Email request password change test fixtures
export const emailRequestPasswordChangeFixtures: ModelTestFixtures<EmailRequestPasswordChangeInput, never> = {
    minimal: {
        create: {
            email: "reset@example.com",
        },
        update: null as never,
    },
    complete: {
        create: {
            email: "complete.reset@example.com",
        },
        update: null as never,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing email
            } as EmailRequestPasswordChangeInput,
            update: null as never,
        },
        invalidTypes: {
            create: {
                email: 123, // Should be string
            } as unknown as EmailRequestPasswordChangeInput,
            update: null as never,
        },
        invalidEmail: {
            create: {
                email: "not-an-email",
            },
            update: null as never,
        },
    },
    edgeCases: {
        emptyEmail: {
            create: {
                email: "",
            },
            update: null as never,
        },
        longEmail: {
            create: {
                email: `${"a".repeat(200)}@example.com`,
            },
            update: null as never,
        },
    },
};

// Email reset password test fixtures
export const emailResetPasswordFixtures: ModelTestFixtures<EmailResetPasswordInput, never> = {
    minimal: {
        create: {
            id: validIds.id1,
            code: "resetcode123",
            newPassword: "NewSecurePassword123!",
        },
        update: null as never,
    },
    complete: {
        create: {
            id: validIds.id2,
            code: "completecode456",
            newPassword: "CompleteNewPassword123!",
        },
        update: null as never,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing required fields
            } as EmailResetPasswordInput,
            update: null as never,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
                code: true, // Should be string
                newPassword: 456, // Should be string
            } as unknown as EmailResetPasswordInput,
            update: null as never,
        },
        weakPassword: {
            create: {
                id: validIds.id1,
                code: "validcode",
                newPassword: "weak",
            },
            update: null as never,
        },
    },
    edgeCases: {
        withPublicId: {
            create: {
                publicId: "user123",
                code: "publiccode",
                newPassword: "PublicIdPassword123!",
            },
            update: null as never,
        },
        emptyCode: {
            create: {
                id: validIds.id1,
                code: "",
                newPassword: "ValidPassword123!",
            },
            update: null as never,
        },
    },
};

// Validate session test fixtures
export const validateSessionFixtures: ModelTestFixtures<ValidateSessionInput, never> = {
    minimal: {
        create: {
            timeZone: "UTC",
        },
        update: null as never,
    },
    complete: {
        create: {
            timeZone: "America/New_York",
        },
        update: null as never,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing timeZone
            } as ValidateSessionInput,
            update: null as never,
        },
        invalidTypes: {
            create: {
                timeZone: 123, // Should be string
            } as unknown as ValidateSessionInput,
            update: null as never,
        },
        invalidTimeZone: {
            create: {
                timeZone: "Invalid/TimeZone",
            },
            update: null as never,
        },
    },
    edgeCases: {
        emptyTimeZone: {
            create: {
                timeZone: "",
            },
            update: null as never,
        },
        longTimeZone: {
            create: {
                timeZone: "Very/Long/Invalid/TimeZone/Path/That/Exceeds/Normal/Limits",
            },
            update: null as never,
        },
        commonTimeZones: {
            create: {
                timeZone: "Europe/London",
            },
            update: null as never,
        },
    },
};

// Switch current account test fixtures
export const switchCurrentAccountFixtures: ModelTestFixtures<SwitchCurrentAccountInput, never> = {
    minimal: {
        create: {
            id: validIds.id1,
        },
        update: null as never,
    },
    complete: {
        create: {
            id: validIds.id2,
        },
        update: null as never,
    },
    invalid: {
        missingRequired: {
            create: {
                // Missing id
            } as SwitchCurrentAccountInput,
            update: null as never,
        },
        invalidTypes: {
            create: {
                id: 123, // Should be string
            } as unknown as SwitchCurrentAccountInput,
            update: null as never,
        },
        invalidId: {
            create: {
                id: "invalid-id-format",
            },
            update: null as never,
        },
    },
    edgeCases: {
        emptyId: {
            create: {
                id: "",
            },
            update: null as never,
        },
        longId: {
            create: {
                id: "999999999999999999999999999999999999999999999999999999999999999999",
            },
            update: null as never,
        },
        shortId: {
            create: {
                id: "1",
            },
            update: null as never,
        },
    },
};