import * as Serialization from '@emurgo/cardano-serialization-lib-nodejs';
import * as MessageSigning from './message_signing/rust/pkg/emurgo_message_signing';

export const generateNonce = (length: number = 64) => {
    if (length <= 0 || length > 2048) throw new Error('Length must be bewteen 1 and 2048');
    require('crypto').randomBytes(48, function(_err: any, buffer: any) {
        return buffer.toString('hex');
    });
}

/**
 * Determines if a wallet address signed a message (payload)
 * @param address Serialized wallet address
 * @param payload Serialized payload (i.e. message with nonce)
 * @param coseSign1Hex Hex string of signed payload (signed by user's wallet)
 * @returns True if payload was signed by wallet address
 */
export const verifySignedMessage = (address: string, payload: string, coseSign1Hex: string) => {
    const coseSign1 = MessageSigning.COSESign1.from_bytes(Buffer.from(coseSign1Hex, 'hex'));
    const payloadCose: Uint8Array | undefined = coseSign1.payload();

    if (!payloadCose || !verifyPayload(payload, payloadCose)) {
        throw new Error('Payload does not match');
    }

    const protectedHeaders: MessageSigning.HeaderMap = coseSign1
        .headers()
        .protected()
        .deserialized_headers();
    const headerCBORBytes: Uint8Array | undefined = protectedHeaders.header(MessageSigning.Label.new_text('address'))?.as_bytes();
    if (!headerCBORBytes) {
        throw new Error('Failed to convert header to bytes');
    }
    const keyId: Uint8Array | undefined = protectedHeaders.key_id();
    if (!keyId) {
        throw new Error('Failed to get keyId from header');
    }
    const addressCose: Serialization.Address = Serialization.Address.from_bytes(headerCBORBytes);
    const publicKeyCose = Serialization.PublicKey.from_bytes(keyId);

    if (!verifyAddress(address, addressCose, publicKeyCose))
        throw new Error('Could not verify because of address mismatch');

    const signature = Serialization.Ed25519Signature.from_bytes(coseSign1.signature());
    const data = coseSign1.signed_data().to_bytes();
    return publicKeyCose.verify(data, signature);
};

const verifyPayload = (payload: string, payloadCose: Uint8Array) => {
    return Buffer.from(payloadCose).compare(Buffer.from(payload, 'hex'));
};

const verifyAddress = (address: string, addressCose: Serialization.Address, publicKeyCose: Serialization.PublicKey) => {
    const checkAddress = Serialization.Address.from_bytes(Buffer.from(address, 'hex'));
    if (addressCose.to_bech32() !== checkAddress.to_bech32()) return false;
    // check if BaseAddress
    try {
        const baseAddress: Serialization.BaseAddress | undefined = Serialization.BaseAddress.from_address(addressCose);
        if (!baseAddress) {
            throw new Error('Failed to get base address from addressCose');
        }
        //reconstruct address
        const paymentKeyHash = publicKeyCose.hash();
        const stakeKeyHash: Serialization.Ed25519KeyHash | undefined = baseAddress.stake_cred().to_keyhash();
        if (!stakeKeyHash) {
            throw new Error('Failed to find stake key hash');
        }
        const reconstructedAddress = Serialization.BaseAddress.new(
            checkAddress.network_id(),
            Serialization.StakeCredential.from_keyhash(paymentKeyHash),
            Serialization.StakeCredential.from_keyhash(stakeKeyHash)
        );
        if (
            checkAddress.to_bech32() !== reconstructedAddress.to_address().to_bech32()
        )
            return false;

        return true;
    } catch (error) {
        console.error('Caught error verifying address', error);
    }
    // check if RewardAddress
    try {
        //reconstruct address
        const stakeKeyHash = publicKeyCose.hash();
        const reconstructedAddress = Serialization.RewardAddress.new(
            checkAddress.network_id(),
            Serialization.StakeCredential.from_keyhash(stakeKeyHash)
        );
        if (
            checkAddress.to_bech32() !== reconstructedAddress.to_address().to_bech32()
        )
            return false;

        return true;
    } catch (error) {
        console.error('Caught error verifying address', error)
    }
    return false;
};
