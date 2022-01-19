// Handles wallet integration
import { walletInitMutation, walletCompleteMutation } from 'graphql/mutation';
import { initializeApollo } from 'graphql/utils/initialize';
import { Session } from 'types';
import { Pubs } from 'utils';

enum WalletProvider {
    Nami
}
/**
 * Maps supported wallet providers to their key name in window.cardano. 
 * See CIP-0030 for more info: https://github.com/cardano-foundation/CIPs/pull/148
 */
const Provider = {
    [WalletProvider.Nami]: 'nami'
}

/**
 * Maps network names to their ids
 */
const Network = {
    Mainnet: 1,
    Testnet: 0
}

export const hasWalletExtension = () => Boolean(window.cardano);

/**
 * Connects to wallet provider
 * @param provider The wallet provider to connect to
 * @returns Object containing methods to interact with the wallet provider
 */
const connectWallet = async (provider: WalletProvider): Promise<any> => {
    if (!hasWalletExtension()) return false;
    return await window.cardano[Provider[provider]].enable();
}

// Initiate handshake to verify wallet with backend
// Returns hex string of payload, to be signed by wallet
const walletInit = async (publicAddress: string): Promise<any> => {
    const client = initializeApollo();
    const result = await client.mutate({
        mutation: walletInitMutation,
        variables: { input: { publicAddress } }
    });
    return result.data.walletInit;
}

/**
 * Completes handshake to verify wallet with backend
 * @param publicAddress Wallet's public address
 * @param message Message signed by wallet
 * @returns Session object if successful, null if not
 */
const walletComplete = async (publicAddress: string, message: { key: string, signature: string }): Promise<Session | null> => {
    const client = initializeApollo();
    console.log('in wallet complete', message);
    const result = await client.mutate({
        mutation: walletCompleteMutation,
        variables: { input: { publicAddress, ...message } }
    });
    return result.data.walletComplete;
}

// Signs payload received from walletInit
const signPayload = async (walletActions: any, publicAddress: string, payload: string): Promise<any> => {
    if (!hasWalletExtension()) return null;
    return await walletActions.signData(publicAddress, payload);
}

/**
 * Establish trust between a user's wallet and the backend
 * @returns Session object or null
 */
export const validateWallet = async (provider: WalletProvider = WalletProvider.Nami): Promise<Session | null> => {
    let session: Session | null = null;
    try {
        // Connect to wallet extension
        const walletActions = await connectWallet(provider);
        if (!walletActions) return null;
        console.log('walletActions', walletActions);
        // Check if wallet is mainnet or testnet
        const network = await walletActions.getNetworkId();
        if (network !== Network.Mainnet) throw new Error('Wallet is not on mainnet');
        // Find wallet address
        const rewardAddresses = await walletActions.getRewardAddresses();
        if (!rewardAddresses || rewardAddresses.length === 0) throw new Error('Could not find reward address');
        console.log('rewardAddress', rewardAddresses[0]);
        // Request payload from backend
        const payload = await walletInit(rewardAddresses[0]);
        if (!payload) return null;
        // Sign payload with wallet
        const signedPayload = await signPayload(walletActions, rewardAddresses[0], payload);
        console.log('signed payload', signedPayload);
        if (!signedPayload) return null;
        // Send signed payload to backend for verification
        session = await walletComplete(rewardAddresses[0], signedPayload);
    } catch (error: any) {
        console.error('Caught error completing wallet validation', error);
        PubSub.publish(Pubs.AlertDialog, {
            message: error.message ?? 'Unknown error occurred',
            buttons: [{ text: 'OK' }]
        });
    } finally {
        return session;
    }
}