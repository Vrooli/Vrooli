/**
 * Finds the name of the device running the app, 
 * using navigator
 * @returns The device's name, or undefined if it can't be found
 */
export const getDeviceName = (): string | undefined => {
    // Get the user agent string
    const userAgent = navigator.userAgent;

    // Use a regular expression to extract the device name from the user agent string
    const deviceNameRegex = /\((.+?)\)/;
    const deviceNameMatch = userAgent.match(deviceNameRegex);
    return deviceNameMatch ? deviceNameMatch[1] : undefined;
}