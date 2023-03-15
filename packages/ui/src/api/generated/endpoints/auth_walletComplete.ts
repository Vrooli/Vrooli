import gql from 'graphql-tag';
import { Label_full } from '../fragments/Label_full';
import { Label_list } from '../fragments/Label_list';
import { Session_full } from '../fragments/Session_full';
import { Wallet_common } from '../fragments/Wallet_common';

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

