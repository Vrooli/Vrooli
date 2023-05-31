import gql from "graphql-tag";
import { Schedule_common } from "../fragments/Schedule_common";
import { Session_full } from "../fragments/Session_full";
import { Wallet_common } from "../fragments/Wallet_common";

export const authWalletComplete = gql`${Schedule_common}
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

