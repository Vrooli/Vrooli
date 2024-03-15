import pkg from "@prisma/client";
import { gql } from "apollo-server-express";
import { AuthEndpoints, EndpointsAuth } from "../logic/auth";

const { AccountStatus } = pkg;

export const typeDef = gql`
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

    input LogOutInput {
        id: ID
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

    type SessionUser {
        activeFocusMode: ActiveFocusMode
        apisCount: Int!
        bookmarkLists: [BookmarkList!]! # Will not include the bookmarks themselves, just info about the lists
        credits: String! # Stringified BigInt
        focusModes: [FocusMode!]!
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
        smartContractsCount: Int!
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
        logOut(input: LogOutInput!): Session!
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
