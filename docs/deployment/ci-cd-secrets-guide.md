# Setting Up Environment File Secrets for CI/CD

This guide explains how to set up GitHub Actions secrets for your Vrooli CI/CD pipelines.

## Required Secrets

### Environment File Secrets
These secrets are used to create environment files during the CI/CD process:

1. `ENV_DEV_FILE`/`ENV_PROD_FILE` - Contains the entire contents of the `.env-dev`/`.env-prod` file (for development/production deployments)

These secrets are used for deploying to the development environment:

1. `DEV_SSH_PRIVATE_KEY`/`SSH_PRIVATE_KEY` - SSH private key for connecting to the development/production server
2. `DEV_VPS_HOST`/`VPS_HOST` - Hostname or IP address of the development/production server
3. `DEV_VPS_USER`/`VPS_USER` - Username for SSH access to the development/production server
4. `DEV_API_URL`/`API_URL` - API URL for health check verification after development/production deployment

### Notification Secrets
1. `NOTIFICATION_WEBHOOK_URL` - Webhook URL for deployment notifications

## Creating Environment File Secrets

### Step 1: Prepare Your Environment Files

1. Ensure your local `.env-dev` and `.env-prod` files contain all required variables

### Step 2: Add Secrets to GitHub

1. Navigate to your GitHub repository
2. Go to Settings → Secrets and variables → Actions
3. Click "Manage repository secrets"
4. Click or add target environment ("development" or "production")
5. For each secret, click "Add repository secret" and enter the secret name and value
6. For each variable, click "Add variable" and enter the variable name and value

### Step 3: Verify Secret Setup

1. Navigate to the "Actions" tab in your repository
2. For the most recent workflow run, check the logs for any environment-related errors
3. If you see `Could not find .env-dev file at /home/runner/work/Vrooli/Vrooli/scripts/../.env-dev. Exiting...`, the secret is not set up correctly

### For SSH Access Secrets

Both development and production workflows require SSH access:

1. Generate an SSH key pair for each environment if you haven't already:
   ```bash
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_vrooli_dev
   ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa_vrooli_prod
   ```
2. Add the public key to the authorized_keys file on each server
3. Add the private key as a secret in GitHub:
   - For development: `DEV_SSH_PRIVATE_KEY`
   - For production: `SSH_PRIVATE_KEY`

## Updating Secrets

When you need to update environment variables:

1. Go to Settings → Secrets and variables → Actions
2. Find the appropriate secret
3. Click "Update"
4. Paste the new content
5. Click "Update secret"

## Environment Protection Rules

For production deployments, consider setting up environment protection rules:

1. Go to Settings → Environments
2. Click on "production"
3. Set up required reviewers for production deployments
4. Optionally add environment variables specific to the production environment

## Security Considerations

- These secrets contain sensitive information like API keys and database credentials
- GitHub securely encrypts these secrets and only exposes them during workflow runs
- The secrets are never shown in logs
- Consider using GitHub's environment protection rules for the production environment

## Troubleshooting

If you encounter issues with the environment file creation:

1. Check the workflow run logs for any errors
2. Ensure the secret name exactly matches what's referenced in the workflow file
3. Verify the secret contains the entire environment file, including all required variables
4. Make sure there are no extra spaces or newlines at the beginning or end of the secret value

### SSH Connection Issues

If you have SSH connection problems:
1. Verify the SSH key has been properly added to the server
2. Check that the hostname/IP address (`VPS_HOST` or `DEV_VPS_HOST`) is correct
3. Ensure the username (`VPS_USER` or `DEV_VPS_USER`) has proper permissions
4. Confirm the SSH key format in the GitHub secret (should include headers and footers) 