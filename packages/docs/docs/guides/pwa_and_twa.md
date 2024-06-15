# PWAs and TWAs

## Progressive Web App (PWA)
A PWA is a website that can be installed on mobile devices. These don't have quite the same functionality as native apps, but hopefully one day they will. To make your website PWA-compatable, perform an audit on Lighthouse (found in your browser's developer console). Then, follow the steps it provides. **Make sure `NODE_ENV` is set to `production` when testing PWA.**

## App Store Eligibility
Use the [PWABuilder tool](https://www.pwabuilder.com/) to check if your PWA is eligible for various app stores. This website generates a score for your PWA based on the information provided in your manifest file, the service worker, and other factors.

## Trusted Web Activity (TWA)
A trusted web activity is a PWA that runs natively on Android devices. They can also be listed on the Google Play store, making them almost identical to traditional apps. 

### Creating a TWA
First, make sure that the `packages/ui/public/site.manifest` or `packages/ui/public/manifest.json` file has the following data:  
1. orientation and display to define how the app should feel (likely "any" and "standalone" to feel like a native app)
2. screenshots (displayed in the store)  
3. name, short name, and description   
4. dir (direction of text) and lang for localization
5. icons
6. start_url and scope
, along with everything else mentioned in the Favicons and PWA sections of this guide. All known manifest fields can be found [here](https://developer.mozilla.org/en-US/docs/Web/Manifest/categories).

Once that is complete, you can use the [PWABuilder tool](https://www.pwabuilder.com/) to generate an Android package. Make sure the following fields are filled out (you may need to press "All Settings"):
- Package ID: Should match what's stored in the `target.package_name` field when we create `assetlinks.json` in the `build.sh` script. Typically `com.vrooli.twa`.
- Version: The current version in the `package.json` file
- Version code: Any integer higher than the previous version code. A good approach if you don't feel like checking the previous code is to use the version without decimals and add zeros until it's 4 digits. For example, if the version is 1.2.3, the version code should be 1230.
- Signing Key: Select the "Use mine" option
    - Key file: Use the *.jks file generated from `build.sh`
    - Key alias: What's defined in the `build.sh` script (should be "upload" by default)
    - Key password: The `GOOGLE_PLAY_KEYSTORE_PASSWORD` defined in the `.env` file
    - Key store password: The same as the key password

If all goes well, you should receive a zip file with the TWA package. Unzip it to receive a `.aab` file and a `.apk` file. The first is what gets uploaded to the Google Play store, while the second is for testing purposes.

### Testing a TWA
To test your TWA locally, you can use either Android Studio emulators or a physical Android device:

#### Using Android Studio Emulator
1. Install and open Android Studio.
2. Navigate to the **Virtual Device Manager** and set up an Android emulator.
3. Start the emulator.
4. Drag and drop the `.apk` file onto the emulator to install
5. Open the app from the app drawer in the emulator to test its functionality.

#### Using a Physical Android Device
1. Enable Developer Options and USB Debugging on your Android device.
2. Connect your device to your computer via USB.
3. Transfer the `.apk` file to your device and install it, or use:
   ```bash
   adb install path_to_your_apk/app-release.apk
   ```
4. Open the app from the device's app drawer to ensure it functions as expected.

### Deploying a TWA
When you're ready to test or deploy your app to the Google Play store, visit [Google Play Console](https://play.google.com/) to receive instructions.