/** 
 * Regular expression for validating links.
 * URL (taken from here: https://stackoverflow.com/a/9284473/10240279)
 */
export const urlRegex = /^(?:(?:https?|ftp):\/\/)(?:\S+(?::\S*)?@)?(?:(?!(?:10|127)(?:\.\d{1,3}){3})(?!(?:169\.254|192\.168)(?:\.\d{1,3}){2})(?!172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?:[1-9]\d?|1\d\d|2[01]\d|22[0-3])(?:\.(?:1?\d{1,2}|2[0-4]\d|25[0-5])){2}(?:\.(?:[1-9]\d?|1\d\d|2[0-4]\d|25[0-4]))|(?:(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)(?:\.(?:[a-z\u00a1-\uffff0-9]-*)*[a-z\u00a1-\uffff0-9]+)*(?:\.(?:[a-z\u00a1-\uffff]{2,}))\.?)(?::\d{2,5})?(?:[/?#]\S*)?$/i;

/** Regular expression for validating links in development mode (relaxed) */
export const urlRegexDev = /^(?:(?:https?|ftp):\/\/)(?:\S+(?::\S*)?@)?(?:(?:localhost|127\.0\.0\.1)(?::\d{2,5})?|(?:[a-z\u00a1-\uffff0-9]+(?:-[a-z\u00a1-\uffff0-9]+)*)(?:\.(?:[a-z\u00a1-\uffff0-9]+(?:-[a-z\u00a1-\uffff0-9]+)*))*\.(?:[a-z\u00a1-\uffff]{2,})\.?)(?::\d{2,5})?(?:[/?#]\S*)?$/i;

/** Cardano payment wallet address (i.e. starts with "addr1", and is 103 characters long in total) */
export const walletAddressRegex = /^addr1[a-zA-Z0-9]{98}$/;

/** Handle (i.e. 3-16 characters, no special characters except underscores) */
export const handleRegex = /^(?=.*[a-zA-Z0-9])[a-zA-Z0-9_]{3,16}$/;

/** Hex code */
export const hexColorRegex = /^#([0-9A-F]{3}([0-9A-F]{3})?)$/i;
