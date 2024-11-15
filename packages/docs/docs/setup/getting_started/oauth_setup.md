# OAuth Implementation Guide

OAuth (Open Authorization) is an open standard for access delegation, commonly used as a way for users to grant websites or applications access to their information on other websites without giving them their passwords. It enables secure and streamlined user authentication by allowing users to log in using their existing accounts from providers like Google, Facebook, GitHub, etc.

Currently, we use OAuth to simplify the authentication process for users, enhancing both security and user experience. This guide will show you how to implement OAuth in this application.

## Table of Contents

1. [Understanding OAuth](#understanding-oauth)
2. [Why Use OAuth?](#why-use-oauth)
3. [Step 1: Choose an OAuth Provider](#step-1-choose-an-oauth-provider)
4. [Step 2: Register Your Application with the Provider](#step-2-register-your-application-with-the-provider)
5. [Step 3: Store Credentials in .env and .env-prod](#step-3-store-credentials-in-env-and-env-prod)
6. [Step 4: Set Up the Server for OAuth](#step-4-set-up-the-server-for-oauth)
7. [Step 5: Update the OAUTH_PROVIDERS List in LoginView.tsx](#step-5-update-the-oauth_providers-list-in-loginviewtsx)
8. [Additional Considerations](#additional-considerations)

---

## Understanding OAuth

OAuth provides a secure method for users to grant limited access to their resources on one site to another site, without having to expose their credentials. It works by allowing the user to authorize an application to access their data hosted with another service provider.

## Why Use OAuth?

- **Enhanced Security**: Users do not have to share their passwords with third-party applications.
- **Improved User Experience**: Users can log in quickly using accounts they already have.
- **Trustworthiness**: Leveraging reputable providers increases user trust in the authentication process.

---

## Step 1: Choose an OAuth Provider

Select the OAuth providers you wish to support. Common providers include:

- **X**
- **Google**
- **Facebook**
- **GitHub**
- **Twitter**
- **LinkedIn**

Each provider requires you to register your application to obtain credentials.

---

## Step 2: Register Your Application with the Provider

**Note 1:** Most OAuth providers require you to officially publish your application before OAuth works outside of the development environment. If this is your first time using OAuth, you may want to test with a single provider first before setting up all of them.

**Note 2:** The less information we need from the user, the better. Vrooli only needs the user's email and name. Keep this in mind when setting the OAuth scopes.

**Note 3:** You will need to set up different credentials for your development and production environments. Development should use localhost or your staging server domain, while production should use your live domain.

### Example: Registering with Google

1. **Visit the Google API Console**
   - Go to [Google API Console](https://console.developers.google.com/).

2. **Create a New Project**
   - Click on the project drop-down and select **"New Project"**.
   - Enter a **Project Name** and click **"Create"**.

3. **Configure the OAuth Consent Screen**
   - Navigate to **"OAuth consent screen"** in the sidebar.
   - Choose **"External"**.
   - Fill out the required fields (app name, user support email, etc.).
   - Save your changes.

4. **Add Scopes**
    - Click on "Add or remove scopes".
    - Select the following scopes:
        - email: View your email address (.../auth/userinfo.email).
        - profile: See your personal info, including any personal info you've made publicly available (.../auth/userinfo.profile).

5. **Create OAuth Client IDs**
   - Go to **"Credentials"** in the sidebar.
   - Click **"Create credentials"** and select **"OAuth client ID"**.
   - Choose the application type (e.g., Web application).
   - Enter a **Name** for the client.
       - **Note:** A good naming convention is to include the application type, environment, and date (e.g. `Web client - localhost - 20241113`).
   - Enter the site's URL for **Authorized JavaScript origins** (e.g., `http://localhost:3000`).
   - Enter the redirect URI for **Authorized redirect URIs** (e.g., `http://localhost:3000/auth/google/callback`).
   - Securely store the generated client ID and client secret.

   Repeat for each application type (Web, Android, iOS, etc.).

*Note: The steps are similar for other providers. Refer to their documentation for specifics.*

---

## Step 3: Store Credentials in .env and .env-prod

1. **Open `.env` or `.env-prod`** file in your project's root directory.

2. **Add Your OAuth Credentials**
   - For Google:

     ```env
     GOOGLE_CLIENT_ID=your-google-client-id
     GOOGLE_CLIENT_SECRET=your-google-client-secret
     ```

   - For other providers, use their respective environment variables:

     ```env
     FACEBOOK_APP_ID=your-facebook-app-id
     FACEBOOK_APP_SECRET=your-facebook-app-secret
     ```

---

## Step 4: Set Up the Server for OAuth

You'll need to configure the server to handle OAuth authentication flows. We're currently using `passport.js` for this purpose, which includes various strategies for different OAuth providers (and also non-OAuth methods like local authentication).

If the desired OAuth provider is not already supported in our server, go to [the passport.js strategies search page](https://www.passportjs.org/packages/) to find the appropriate strategy. Each strategy requires its own package installation and setup.

All passport strategies and callback routes are defined in the `passport.ts` file.

---

## Step 5: Update the `OAUTH_PROVIDERS` List in `LoginView.tsx`

1. **Locate `LoginView.tsx`**

   - Navigate to `src/components/LoginView.tsx`.

2. **Update the `OAUTH_PROVIDERS` Array**

   ```typescript
   // LoginView.tsx

   const OAUTH_PROVIDERS = [
     {
       name: 'Google',
       url: '/auth/google',
       icon: 'path-to-google-icon.png',
     },
     {
       name: 'Facebook',
       url: '/auth/facebook',
       icon: 'path-to-facebook-icon.png',
     },
     // Add more providers as needed
   ];
   ```

4. **Ensure Assets Are Available**

   - Place provider icons in the appropriate directory and ensure the paths in `icon` are correct.

---

*Refer to the official documentation of each OAuth provider for more detailed information and assistance.*

- [Google OAuth Documentation](https://developers.google.com/identity/protocols/oauth2)
- [Facebook Login Documentation](https://developers.facebook.com/docs/facebook-login/)
- [Passport.js Documentation](http://www.passportjs.org/docs/)

---

By following this guide, you should have a working OAuth authentication system integrated into your application. Remember to keep your OAuth credentials secure and to comply with each provider's policies.