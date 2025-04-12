# OAuth Setup Guide

This guide will help you set up OAuth authentication for your Vrooli application, allowing users to sign in with popular social media platforms.

## Overview

OAuth authentication enables users to sign in to your application using their existing accounts from providers like Google, GitHub, or Facebook. This improves the user experience by eliminating the need to create and remember another set of credentials.

## Prerequisites

Before you begin, make sure you have:
- Completed the [Repo Setup](repo_setup.md)
- A domain name (for production setup)
- Access to update environment variables

## Google OAuth Setup

### 1. Create a Google Cloud Project

1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Navigate to "APIs & Services" > "OAuth consent screen"
4. Select the appropriate user type (External or Internal)
5. Fill in the required application information:
   - App name
   - User support email
   - Developer contact information
6. Add the following scopes:
   - `./auth/userinfo.email`
   - `./auth/userinfo.profile`
7. Add your domain to the "Authorized domains" list
8. Save and continue

### 2. Create OAuth Credentials

1. Go to "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth client ID"
3. Select "Web application" as the application type
4. Give your client a name (e.g., "Vrooli Web Client")
5. Add the following to "Authorized JavaScript origins":
   - For development: `http://localhost:3000`
   - For production: `https://yourdomain.com`
6. Add the following to "Authorized redirect URIs":
   - For development: `http://localhost:3000/api/auth/callback/google`
   - For production: `https://yourdomain.com/api/auth/callback/google`
7. Click "Create"
8. Note the "Client ID" and "Client Secret"

### 3. Configure Environment Variables

Add the following to your `.env` file:

```
# Google OAuth
GOOGLE_CLIENT_ID="your_client_id"
GOOGLE_CLIENT_SECRET="your_client_secret"
```

## GitHub OAuth Setup

### 1. Register a New OAuth Application

1. Go to your GitHub account settings
2. Navigate to "Developer settings" > "OAuth Apps"
3. Click "New OAuth App"
4. Fill in the application details:
   - Application name (e.g., "Vrooli")
   - Homepage URL (e.g., `http://localhost:3000` for development or `https://yourdomain.com` for production)
   - Authorization callback URL (e.g., `http://localhost:3000/api/auth/callback/github` for development)
5. Click "Register application"
6. Note the "Client ID"
7. Generate a new client secret and note it down (you won't be able to see it again)

### 2. Configure Environment Variables

Add the following to your `.env` file:

```
# GitHub OAuth
GITHUB_CLIENT_ID="your_client_id"
GITHUB_CLIENT_SECRET="your_client_secret"
```

## Facebook OAuth Setup

### 1. Create a Facebook App

1. Go to the [Facebook Developers](https://developers.facebook.com/) site
2. Click "My Apps" > "Create App"
3. Select the app type "Consumer" and click "Next"
4. Fill in the app name and contact email, then click "Create App"
5. In the dashboard, add the "Facebook Login" product
6. Configure the "Facebook Login" settings:
   - Under "Settings" > "Basic", note the "App ID" and "App Secret"
   - Under "Facebook Login" > "Settings", add the following to "Valid OAuth Redirect URIs":
     - For development: `http://localhost:3000/api/auth/callback/facebook`
     - For production: `https://yourdomain.com/api/auth/callback/facebook`

### 2. Configure Environment Variables

Add the following to your `.env` file:

```
# Facebook OAuth
FACEBOOK_CLIENT_ID="your_app_id"
FACEBOOK_CLIENT_SECRET="your_app_secret"
```

## Testing Your OAuth Setup

1. Restart your application
2. Visit your login page
3. Attempt to sign in with each configured OAuth provider
4. Verify that:
   - You're redirected to the provider's authentication page
   - After authenticating, you're redirected back to your application
   - You're successfully signed in with the correct user information

## Production Considerations

When moving to production:

1. Update all redirect URIs to use your production domain
2. Ensure your OAuth consent screen is configured for production if using Google
3. Make sure your environment variables are updated for the production environment
4. Consider implementing additional security measures like CSRF protection

## Troubleshooting

If you encounter issues:

1. Check your server logs for detailed error messages
2. Verify that your redirect URIs exactly match what's configured in the provider's dashboard
3. Ensure your client IDs and secrets are correctly set in your environment variables
4. For Google, make sure your application is properly verified if you're using it in production