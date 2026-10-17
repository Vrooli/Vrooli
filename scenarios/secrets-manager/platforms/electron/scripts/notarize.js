const { notarize } = require('@electron/notarize');

exports.default = async function notarizing(context) {
    const { electronPlatformName, appOutDir } = context;
    if (electronPlatformName !== 'darwin') {
        return;
    }

    const appName = context.packager.appInfo.productFilename;

    return await notarize({
        appBundleId: 'com.vrooli.secrets-manager',
        appPath: `${appOutDir}/${appName}.app`,
        appleId: process.env.APPLE_ID,
        // Keep the template generator aligned with the shared desktop
        // credential declaration. The API generator accepts a custom env
        // binding, while this legacy template uses the canonical default.
        appleIdPassword: process.env.APPLE_APP_SPECIFIC_PASSWORD,
        teamId: process.env.APPLE_TEAM_ID,
    });
};