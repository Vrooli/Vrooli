export enum DeviceType {
    Mobile = "Mobile",
    Desktop = "Desktop",
}

export enum DeviceOS {
    Android = "Android",
    IOS = "iOS",
    Windows = "Windows",
    MacOS = "MacOS",
    Linux = "Linux",
    Unknown = "Unknown",
}

/**
 *  Windows keys which differ on other operating systems
 */
export enum WindowsKey {
    Ctrl = "Ctrl",
    Alt = "Alt",
    Enter = "Enter",
}

/**
 * Windows to Mac key mapping
 */
export enum MacKeyFromWindows {
    Ctrl = "⌘",
    Alt = "⌥",
    Enter = "↩",
}

/**
 * All keys allowed in a key combination
 */
export type KeyComboOption = `${WindowsKey}` | "Shift" | "Tab" | "Backspace" | "Delete" | "Escape" | "Space" | "ArrowUp" | "ArrowDown" | "ArrowLeft" | "ArrowRight" | "Home" | "End" | "PageUp" | "PageDown" | "Insert" | "F1" | "F2" | "F3" | "F4" | "F5" | "F6" | "F7" | "F8" | "F9" | "F10" | "F11" | "F12" | "A" | "B" | "C" | "D" | "E" | "F" | "G" | "H" | "I" | "J" | "K" | "L" | "M" | "N" | "O" | "P" | "Q" | "R" | "S" | "T" | "U" | "V" | "W" | "X" | "Y" | "Z" | "0" | "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9";

/**
 * Finds the device name, type, and operating system using navigator. 
 * Any values not found will be undefined.
 * NOTE: This data can be spoofed by the user
 */
export const getDeviceInfo = (): {
    deviceName: string | undefined,
    deviceType: DeviceType | undefined,
    deviceOS: DeviceOS | undefined,
    isStandalone: boolean,
} => {
    // Get the user agent string
    const userAgent = navigator.userAgent;
    // Determine name
    // Use a regular expression to extract the device name from the user agent string
    const deviceNameRegex = /\((.+?)\)/;
    const deviceNameMatch = userAgent.match(deviceNameRegex);
    const deviceName = deviceNameMatch ? deviceNameMatch[1] : undefined;
    // Determine type
    const isMobile = /Mobile|Android|webOS|iPhone|iPad|iPod|BlackBerry|BB|PlayBook|IEMobile|Windows Phone|Kindle|Silk|Opera Mini/i.test(userAgent);
    const deviceType = isMobile ? DeviceType.Mobile : DeviceType.Desktop;
    // Determine OS
    const isAndroid = /Android/i.test(userAgent);
    const isIOS = /iPhone|iPad|iPod/i.test(userAgent);
    const isWindows = /IEMobile|Windows Phone/i.test(userAgent);
    const isMacOS = /Macintosh/i.test(userAgent);
    const isLinux = /Linux/i.test(userAgent);
    const deviceOS = isAndroid ? DeviceOS.Android : isIOS ? DeviceOS.IOS : isWindows ? DeviceOS.Windows : isMacOS ? DeviceOS.MacOS : isLinux ? DeviceOS.Linux : DeviceOS.Unknown;
    // Check if the app is running in standalone mode (i.e. downloaded to the home screen
    const isStandalone = window.matchMedia("(display-mode: standalone)").matches;
    return { deviceName, deviceType, deviceOS, isStandalone };
};

/**
 * Converts a key combination into a string for display.
 * @param keys A list of keys to display
 * @returns A string representation of the key combination
 */
export const keyComboToString = (...keys: KeyComboOption[]): string => {
    // Find the device's operating system
    const { deviceOS } = getDeviceInfo();
    // Initialize the result string
    let result = "";
    // Iterate over the keys
    for (const key of keys) {
        // If the key is a WindowsKey, convert it to the correct key for the device's operating system
        if (key in WindowsKey) {
            result += deviceOS === DeviceOS.MacOS ? MacKeyFromWindows[key] : key;
        } else {
            result += key;
        }
        result += " + ";
    }
    // Remove the trailing ' + '
    result = result.slice(0, -3);
    return result;
};
