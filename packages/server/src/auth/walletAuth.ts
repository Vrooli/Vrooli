import * as Serialization from '@emurgo/cardano-serialization-lib-nodejs';
import * as MessageSigning from './message_signing/rust/pkg/emurgo_message_signing';
import { randomBytes } from 'crypto';
import { genErrorCode, logger, LogLevel } from '../logger';

/**
 * Converts a serialized wallet address to Bech32 format
 * @param address Serialized wallet address
 * @returns Bech32 format of wallet address
 */
export const serializedAddressToBech32 = (address: string) => {
    const addressBytes = Serialization.Address.from_bytes(Buffer.from(address, 'hex'));
    return addressBytes.to_bech32();
}

/**
 * Generate a random string of the specified length, consisting of the specified characters
 * @param length The length of sting to generate
 * @param chars The available characters to use in the string
 * @returns A random string of the specified length, consisting of the specified characters
 */
export function randomString(
    length: number = 64,
    chars: string = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
): string {
    // Check for valid parameters
    if (length <= 0 || length > 2048) throw new Error('Length must be bewteen 1 and 2048.');
    const charsLength = chars.length;
    if (charsLength < 10 || chars.length > 256) throw new Error('Chars must be bewteen 10 and 256.');
    // Generate random bytes
    const bytes = randomBytes(length);
    // Create result array
    let result = new Array(length);
    // Fill result array with bytes, modified to consist of the specified characters
    let cursor = 0;
    for (let i = 0; i < length; i++) {
        cursor += bytes[i];
        result[i] = chars[cursor % charsLength];
    }
    // Return result as string
    return result.join('');
}

/**
 * Generates signable nonce, which includes human-readable description
 * @param description Description to include with nonce, which (depending on the wallet extension) will be displayed to the user
 * @param length Length of nonce to generate
 * @returns Hex string of nonce
 */
export const generateNonce = async (
    description: string = 'Please sign this message so we can verify your wallet:',
    length: number = 64,
) => {
    if (length <= 0 || length > 2048) throw new Error('Length must be bewteen 1 and 2048');
    // Generate nonce (payload)
    const payload = randomString(length);
    // Return description + nonce
    return Buffer.from(`${description} ${payload}`).toString('hex');
}

/**
 * Determines if a wallet address signed a message (payload)
 * @param address Serialized wallet staking address
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
    return Buffer.from(payloadCose).compare(Buffer.from(payload, 'hex')) === 0;
};

const verifyAddress = (address: string, addressCose: Serialization.Address, publicKeyCose: Serialization.PublicKey) => {
    const checkAddress = Serialization.Address.from_bytes(Buffer.from(address, 'hex'));
    if (addressCose.to_bech32() !== checkAddress.to_bech32()) return false;
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
        logger.log(LogLevel.error, 'Caught error validating wallet address', { code: genErrorCode('0005'), error });
    }
    return false;
};
