export var DeviceType;
(function (DeviceType) {
    DeviceType["Mobile"] = "Mobile";
    DeviceType["Desktop"] = "Desktop";
})(DeviceType || (DeviceType = {}));
export var DeviceOS;
(function (DeviceOS) {
    DeviceOS["Android"] = "Android";
    DeviceOS["IOS"] = "iOS";
    DeviceOS["Windows"] = "Windows";
    DeviceOS["MacOS"] = "MacOS";
    DeviceOS["Linux"] = "Linux";
    DeviceOS["Unknown"] = "Unknown";
})(DeviceOS || (DeviceOS = {}));
;
export var WindowsKey;
(function (WindowsKey) {
    WindowsKey["Ctrl"] = "Ctrl";
    WindowsKey["Alt"] = "Alt";
    WindowsKey["Enter"] = "Enter";
})(WindowsKey || (WindowsKey = {}));
export var MacKeyFromWindows;
(function (MacKeyFromWindows) {
    MacKeyFromWindows["Ctrl"] = "\u2318";
    MacKeyFromWindows["Alt"] = "\u2325";
    MacKeyFromWindows["Enter"] = "\u21A9";
})(MacKeyFromWindows || (MacKeyFromWindows = {}));
export const getDeviceInfo = () => {
    const userAgent = navigator.userAgent;
    const deviceNameRegex = /\((.+?)\)/;
    const deviceNameMatch = userAgent.match(deviceNameRegex);
    const deviceName = deviceNameMatch ? deviceNameMatch[1] : undefined;
    const isMobile = /Mobile|Android|webOS|iPhone|iPad|iPod|BlackBerry|BB|PlayBook|IEMobile|Windows Phone|Kindle|Silk|Opera Mini/i.test(userAgent);
    const deviceType = isMobile ? DeviceType.Mobile : DeviceType.Desktop;
    const isAndroid = /Android/i.test(userAgent);
    const isIOS = /iPhone|iPad|iPod/i.test(userAgent);
    const isWindows = /IEMobile|Windows Phone/i.test(userAgent);
    const isMacOS = /Macintosh/i.test(userAgent);
    const isLinux = /Linux/i.test(userAgent);
    const deviceOS = isAndroid ? DeviceOS.Android : isIOS ? DeviceOS.IOS : isWindows ? DeviceOS.Windows : isMacOS ? DeviceOS.MacOS : isLinux ? DeviceOS.Linux : DeviceOS.Unknown;
    const isStandalone = window.matchMedia("(display-mode: standalone)").matches;
    return { deviceName, deviceType, deviceOS, isStandalone };
};
export const keyComboToString = (...keys) => {
    const { deviceOS } = getDeviceInfo();
    let result = "";
    for (const key of keys) {
        if (key in WindowsKey) {
            result += deviceOS === DeviceOS.MacOS ? MacKeyFromWindows[key] : key;
        }
        else {
            result += key;
        }
        result += " + ";
    }
    result = result.slice(0, -3);
    return result;
};
//# sourceMappingURL=device.js.map