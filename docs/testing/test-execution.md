# Test Execution

This document provides guidelines on how to execute various types of tests for the Vrooli platform, including automated tests, linting, and mobile testing setup.

## 1. Running Automated Tests (Unit & Integration)

Automated tests are crucial for verifying the functionality of individual components and their integrations. We use Vitest as our test runner, which includes built-in assertion capabilities and mocking functionality.

### 1.1. Prerequisites

-   Ensure your development environment is set up correctly.
-   Ensure all dependencies are installed (`pnpm install` in the root and relevant package directories).
-   Ensure environment variables are configured (e.g., `.env-test`).

### 1.2. Test Commands

Test commands are typically run from within the specific package directory (e.g., `packages/server` or `packages/shared`). Refer to `docs/tools.md` for a more comprehensive list of project commands.

**Common Test Scripts (from `package.json`):**

-   **Run all tests in a package:**
    ```bash
    # Example for packages/server
    cd packages/server
    pnpm test
    ```
    This command usually builds the test files first and then runs all `*.test.js` files (transpiled from `.ts`) within the `dist` directory.

-   **Run tests in watch mode:**
    ```bash
    # Example for packages/server
    cd packages/server
    pnpm test-watch
    ```
    This will re-run tests automatically when source files change.

-   **Run tests with coverage (example from `packages/shared`):**
    ```bash
    cd packages/shared
    pnpm test-coverage
    # View report (HTML and summary)
    pnpm coverage-report
    ```
    Coverage reports help identify areas of the code not covered by tests. `c8` is used for generating these reports.

-   **Running individual test files:**
    The `docs/tools.md` file provides an example for running a specific test file:
    ```bash
    # Example for /packages/server/src/services/stripe.test.ts
    # (Ensure you are in the packages/server directory)
    clear && pnpm build-tests && npx dotenv -e ../../.env-test -- mocha --file dist/__test/setup.js dist/services/stripe.test.js
    ```
    Adapt the path `dist/services/stripe.test.js` to target the specific compiled test file you want to run.

### 1.3. Interpreting Test Results

-   **Pass:** The test executed successfully and all assertions passed.
-   **Fail:** The test executed, but one or more assertions failed, or an error occurred during test execution.
-   Mocha provides detailed output on failures, including stack traces and assertion differences.

## 2. Linting

Linting is a static analysis process that checks code for programmatic and stylistic errors. It helps maintain code quality and consistency.

-   **Tool:** We use [ESLint](https://eslint.org/) integrated with the [ESLINT VSCode extension](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint).
-   **Configuration:** ESLint is configured via `.eslintrc` files. A main configuration exists at the project root, with package-specific overrides or additions in their respective `.eslintrc` files.
-   **Automatic Linting:** The VSCode extension provides real-time linting for open files.

### 2.1. Manual Linting (Entire Project)

To lint the entire project (or a specific folder) manually:

1.  Open the Command Palette in VSCode (`Ctrl+Shift+P` or `Cmd+Shift+P`).
2.  Type and select `Tasks: Run Task`.
3.  Choose `eslint: lint whole folder` (or a similarly named task if configured).

### 2.2. Disabling/Re-enabling the Linter in VSCode

If the linter impacts performance, you can temporarily disable it:

1.  Open the Command Palette (`Ctrl+Shift+P` or `Cmd+Shift+P`).
2.  Select `Extensions: Focus on Extensions View`.
3.  Search for "ESLint".
4.  Click the gear icon next to the ESLint extension and choose `Disable (Workspace)` or `Disable (Global)`.
5.  To re-enable, follow the same steps and choose `Enable`.

## 3. Mobile Testing (Web App)

This section provides guidance on testing the Vrooli web application on mobile devices, using either emulators or physical devices.

### 3.1. Accessing the Application URL

The URL depends on whether your development server is running locally or on a remote machine. Replace `<UI_PORT>` with the port number specified in your `.env` file (e.g., `3000`).

-   **Remote Server:** Use the server's IP address or domain name (e.g., `http://your-server-ip:<UI_PORT>` or `http://your-domain.com:<UI_PORT>`).
-   **Local Server (Emulator - Android Studio Recommended):**
    -   Use `http://10.0.2.2:<UI_PORT>` from the Android emulator's browser. This IP address is an alias for your host machine's loopback address (127.0.0.1).
-   **Local Server (Physical Device):**
    -   You'll need your computer's local network IP address. Both your computer and the physical mobile device must be on the **same Wi-Fi network**.
    -   **Finding your Local IP Address:**
        -   **Windows (PowerShell/Command Prompt):**
            ```powershell
            # For Wi-Fi connections
            (Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias "Wi-Fi*").IPAddress
            # For Ethernet connections
            (Get-NetIPAddress -AddressFamily IPv4 -InterfaceAlias "Ethernet*").IPAddress
            ```
            Or run `ipconfig` and look for the IPv4 address of your active network adapter.
        -   **macOS (Terminal):**
            ```bash
            ipconfig getifaddr en0  # For Wi-Fi (usually en0 or en1)
            # Or check System Settings > Network > Wi-Fi > Details...
            ```
        -   **Linux (Terminal):**
            ```bash
            hostname -I
            # or
            ip addr show | grep "inet "
            ```
    -   Once you have your computer's IP address (e.g., `192.168.1.100`), the URL for the physical device will be `http://192.168.1.100:<UI_PORT>`.

### 3.2. Using an Emulator (Android Studio Example)

1.  **Install Android Studio:** Download and install from the [official website](https://developer.android.com/studio).
2.  **Set up AVD Manager:** Open Android Studio, go to `Tools > AVD Manager`.
3.  **Create/Start Emulator:** Create a new Android Virtual Device (AVD) with a desired screen size and Android version, or start an existing one.
4.  **Access App:** Open the web browser (e.g., Chrome) within the emulator and navigate to the appropriate URL (see section 3.1).

### 3.3. Using Browser Developer Tools for Mobile Simulation

Most modern desktop browsers (Chrome, Firefox, Edge, Safari) offer built-in developer tools that can simulate various mobile devices, screen sizes, and touch events. This is a quick way to check responsiveness without a physical device or emulator.

-   **Chrome:** Open DevTools (`F12` or `Ctrl+Shift+I`), then click the "Toggle device toolbar" icon (looks like a phone and tablet).
-   Select a device from the dropdown or set custom dimensions.

```mermaid
graph TD
    subgraph Testing Setup
        A[Developer Machine] -->|Runs Dev Server| B(Local Dev Server: localhost:<UI_PORT>)
    end

    subgraph Access Methods
        B --> C{Android Emulator}
        C -->|Browser Navigates To| D[http://10.0.2.2:<UI_PORT>]

        B --> E{Physical Mobile Device}
        A --> |Find Host IP| F(Host IP: e.g., 192.168.1.X)
        F --> E
        E -->|Browser Navigates To| G[http://HOST_IP:<UI_PORT>]
        A -.-> E; # Must be on same Wi-Fi

        H[Remote Dev Server] -->|Browser Navigates To| I[http://SERVER_IP_OR_DOMAIN:<UI_PORT>]
        C --> I
        E --> I
    end
```
This diagram shows how different devices can access the application for testing. 