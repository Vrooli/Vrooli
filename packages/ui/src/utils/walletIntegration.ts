// Handles wallet integration
import { initValidateWalletMutation, completeValidateWalletMutation } from 'graphql/mutation';
import { initializeApollo } from 'graphql/utils/initialize';
import { PUBS } from 'utils';

export const hasWalletExtension = () => Boolean(window.cardano);

const connectWallet = async (): Promise<boolean> => {
    if (!hasWalletExtension()) return false;
    return await window.cardano.enable();
}

// Initiate handshake to verify wallet with backend
// Returns hex string of payload, to be signed by wallet
const initValidateWallet = async (publicAddress: string): Promise<any> => {
    const client = initializeApollo();
    const result = await client.mutate({
        mutation: initValidateWalletMutation,
        variables: { publicAddress }
    });
    return result.data.initValidateWallet;
}

// Signs payload received from initValidateWallet
const signPayload = async (publicAddress: string, payload: string): Promise<any> => {
    if (!hasWalletExtension()) return null;
    return await window.cardano.signData(publicAddress, payload);
}

// Validate payload with backend
export const validateWallet = async (): Promise<any> => {
    let success = false;
    try {
        // Connect to wallet extension
        const walletConnected = connectWallet();
        if (!walletConnected) return false;
        // Find wallet address
        const address = await window.cardano.getRewardAddress();
        // Request payload from backend
        const payload = await initValidateWallet(address);
        if (!payload) return false;
        // Sign payload with wallet
        const signedPayload = await signPayload(address, payload);
        if (!signedPayload) return false;
        // Send signed payload to backend for verification
        const client = initializeApollo();
        client.mutate({
            mutation: completeValidateWalletMutation,
            variables: { publicAddress: address, signedMessage: signedPayload }
        }).then(response => {
            success = response.data.completeValidateWallet;
        })
    } catch (error: any) {
        console.error('Caught error completing wallet validation', error);
        PubSub.publish(PUBS.AlertDialog, {
            message: error.message ?? 'Unknown error occurred',
            buttons: [{ text: 'OK' }]
        });
    } finally {
        return success;
    }
}