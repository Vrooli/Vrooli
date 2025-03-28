# Mobile Testing Setup Guide for React Web App
This guide is intended to assist developers in setting up mobile testing. This is essential to ensure that our app performs well across different devices and platforms. We'll cover both local and remote testing methods.

The app can be testing using a local or remote server, and on a physical or emulated device.

## Finding the URL
The URL of the app depends on whether you're testing on a local or remote server. Whenever you see `<UI_PORT>` in the URL, replace it with the port number specified in your `.env` file.

### Remote Server
For a remote server, the URL can either be the IP address and UI port, or domain name of the server. See the [remote setup guide](/setup/getting_started/remote_setup.html) if you need help setting up a remote server.

### Local Server
For a local server, the URL can be tricky to find. It depends on the type of device you're using for testing.

For an emulator in Android Studio (recommended), simply use `10.0.2.2:<UI_PORT>`.

For a physical device, use the `http://<IP_ADDRESS>:<PORT>`. There are a few ways to find the local IP address:
- In Windows (even if you're using WSL), run `(Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias "Wi-Fi*").IPAddress` in Command Prompt.
- TODO I haven't tested other operating systems yet, so please add instructions for other operating systems if you know how to find the local IP address.

## Using an Emulator (Android Studio)
If you want to use an emulator, one way to do so is through Android Studio:
1. **Open Android Studio**: Start Android Studio and open the AVD Manager to set up an Android emulator.
2. **Create/Start an Emulator**: Create a new virtual device or start an existing one.
3. **Access the App**: In the emulator, open the web browser and enter the URL of the server where the app is hosted. See previous section for details on finding the URL.