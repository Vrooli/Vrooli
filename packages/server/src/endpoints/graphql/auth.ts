import pkg from "@prisma/client";
import { AuthEndpoints, EndpointsAuth } from "../logic/auth";

const { AccountStatus } = pkg;

export const typeDef = `#graphql
    enum AccountStatus {
        Deleted
        Unlocked
        SoftLocked
        HardLocked
    }

    input EmailLogInInput {
        email: String
        password: String
        verificationCode: String
    }

    input EmailSignUpInput {
        name: String!
        email: String!
        theme: String!
        marketingEmails: Boolean!
        password: String!
        confirmPassword: String!
    }

    input EmailRequestPasswordChangeInput {
        email: String!
    }

    input EmailResetPasswordInput {
        id: ID!
        code: String!
        newPassword: String!
    }

    input SwitchCurrentAccountInput {
        id: ID!
    }

    input ValidateSessionInput {
        timeZone: String!
    }

    input WalletInitInput {
        stakingAddress: String!
        nonceDescription: String
    }

    input WalletCompleteInput {
        stakingAddress: String!
        signedPayload: String!
    }

    type WalletComplete {
        firstLogIn: Boolean!
        session: Session
        wallet: Wallet
    }

    type SessionUserSession {
        id: String!
        lastRefreshAt: Date!
    }

    type SessionUser {
        activeFocusMode: ActiveFocusMode
        apisCount: Int!
        codesCount: Int!
        credits: String! # Stringified BigInt
        handle: String
        hasPremium: Boolean!
        id: String!
        languages: [String!]!
        membershipsCount: Int!
        name: String
        notesCount: Int!
        profileImage: String
        projectsCount: Int!
        questionsAskedCount: Int!
        routinesCount: Int!
        session: SessionUserSession!
        standardsCount: Int!
        theme: String
        updated_at: Date!
    }

    type Session {
        isLoggedIn: Boolean!
        timeZone: String
        users: [SessionUser!]
    }

    extend type Mutation {
        emailLogIn(input: EmailLogInInput!): Session!
        emailSignUp(input: EmailSignUpInput!): Session!
        emailRequestPasswordChange(input: EmailRequestPasswordChangeInput!): Success!
        emailResetPassword(input: EmailResetPasswordInput!): Session!
        guestLogIn: Session!
        logOut: Session!
        logOutAll: Session!
        validateSession(input: ValidateSessionInput!): Session!
        switchCurrentAccount(input: SwitchCurrentAccountInput!): Session!
        walletInit(input: WalletInitInput!): String!
        walletComplete(input: WalletCompleteInput!): WalletComplete!
    }
`;

export const resolvers: {
    AccountStatus: typeof AccountStatus;
    Mutation: EndpointsAuth["Mutation"];
} = {
    AccountStatus,
    ...AuthEndpoints,
};
