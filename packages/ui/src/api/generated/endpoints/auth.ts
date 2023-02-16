import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Session_full } from '../fragments/Session_full';
import { Wallet_common } from '../fragments/Wallet_common';

export const authEmailLogIn = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation emailLogIn($input: EmailLogInInput!) {
  emailLogIn(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authEmailSignUp = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation emailSignUp($input: EmailSignUpInput!) {
  emailSignUp(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authEmailRequestPasswordChange = gql`
mutation emailRequestPasswordChange($input: EmailRequestPasswordChangeInput!) {
  emailRequestPasswordChange(input: $input) {
    success
  }
}`;

export const authEmailResetPassword = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation emailResetPassword($input: EmailResetPasswordInput!) {
  emailResetPassword(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authGuestLogIn = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation guestLogIn {
  guestLogIn {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authLogOut = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation logOut($input: LogOutInput!) {
  logOut(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authValidateSession = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation validateSession($input: ValidateSessionInput!) {
  validateSession(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authSwitchCurrentAccount = gql`${Label_full}
${Label_list}
${Organization_nav}
${User_nav}

mutation switchCurrentAccount($input: SwitchCurrentAccountInput!) {
  switchCurrentAccount(input: $input) {
    isLoggedIn
    timeZone
    users {
        handle
        hasPremium
        id
        languages
        name
        schedules {
            filters {
                id
                filterType
                tag {
                    id
                    created_at
                    tag
                    bookmarks
                    translations {
                        id
                        language
                        description
                    }
                    you {
                        isOwn
                        isBookmarked
                    }
                }
                userSchedule {
                    labels {
                        ...Label_list
                    }
                    id
                    name
                    description
                    timeZone
                    eventStart
                    eventEnd
                    recurring
                    recurrStart
                    recurrEnd
                }
            }
            labels {
                ...Label_full
            }
            reminderList {
                id
                created_at
                updated_at
                reminders {
                    id
                    created_at
                    updated_at
                    name
                    description
                    dueDate
                    completed
                    index
                    reminderItems {
                        id
                        created_at
                        updated_at
                        name
                        description
                        dueDate
                        completed
                        index
                    }
                }
            }
            id
            name
            description
            timeZone
            eventStart
            eventEnd
            recurring
            recurrStart
            recurrEnd
        }
        theme
    }
  }
}`;

export const authWalletInit = gql`
mutation walletInit($input: WalletInitInput!) {
  walletInit(input: $input)
}`;

export const authWalletComplete = gql`${Label_full}
${Label_list}
${Session_full}
${Wallet_common}

mutation walletComplete($input: WalletCompleteInput!) {
  walletComplete(input: $input) {
    firstLogIn
    session {
        ...Session_full
    }
    wallet {
        ...Wallet_common
    }
  }
}`;

