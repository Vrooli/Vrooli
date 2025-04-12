# Secrets Management

It is important that secrets are managed properly. Using environment variables in Docker containers can lead to secrets being exposed when images are pushed to Docker Hub. So instead, we store secrets using HashiCorp Vault.

## Introduction to HashiCorp Vault

### What is HashiCorp Vault?
HashiCorp Vault is a tool designed to manage secrets and protect sensitive data. It provides a centralized location to store, access, and manage tokens, passwords, certificates, and encryption keys.

![Vault Architecture](https://www.vaultproject.io/img/vault-architecture.png)

## Setting Up Vault

### Development Setup
Setting up Vault for development is straightforward. In development mode, Vault runs in-memory and is automatically initialized and unsealed.

```bash
vault server -dev
```

See `./scripts/develop.sh` for how we set up Vault in development.

### Production Setup
Production setups require careful planning around storage backends, high availability, and secure access. Steps generally include:

1. Choosing a storage backend.
2. Configuring and starting the Vault server.
3. Initializing and unsealing Vault.

See `./scripts/deploy.sh` for how we set up Vault in production.

## Managing Vault and Accessing the GUI
Vault comes with a built-in web UI:

1. Navigate to your Vault server's address in a web browser. Typically, this is `http://<VAULT_SERVER_ADDRESS>:8200/ui/`.
2. Use your token to login and manage secrets, policies, and other Vault functionalities through the UI.

## Understanding Policies
Policies in Vault define what actions are allowed or disallowed on which paths. They provide a way to grant or deny permissions for Vault operations.

### Managing Policies
Policies are written in HCL (HashiCorp Configuration Language) or JSON. You can create, update, and manage policies via the CLI or the UI.

Example to write a policy:

```bash
vault policy write my-policy my-policy.hcl
```

Example to read a policy:

```bash
vault policy read my-policy
```

Here's what the policy file might look like:

```hcl
# Allow reading and listing secrets for my-app
path "secret/data/my-app/*" {
  capabilities = ["read", "list"]
}

# Allow updating configuration for my-app
path "secret/data/my-app/config" {
  capabilities = ["update"]
}

# Deny access to a specific path
path "secret/data/my-app/creds" {
  capabilities = ["deny"]
}
```

## Giving Access to Vault

### For Virtual Private Servers (VPS)

1. **AppRole**: Use the AppRole authentication method. Assign roles with specific policies. VPS will use `role_id` and `secret_id` to authenticate and obtain tokens.

### For Kubernetes

1. **Kubernetes Auth Method**: Configure Vault to recognize JWT tokens issued by your Kubernetes cluster. Pods authenticate using their service account tokens.

## Secrets in CI/CD Pipelines

When using CI/CD pipelines like GitHub Actions, secrets need to be managed securely. Here are best practices:

### GitHub Secrets for CI/CD

GitHub provides repository-level and organization-level secrets that are encrypted and can be used in workflows:

1. Navigate to your repository settings
2. Go to "Secrets and variables" > "Actions" 
3. Add secrets that your workflows need

These secrets are:
- Encrypted before they reach GitHub
- Masked in logs (hidden from output)
- Not passed to workflows from forked repositories

### Accessing Vault from CI/CD

For more advanced setups, CI/CD pipelines can access Vault directly:

1. **GitHub Actions to Vault**: Configure a GitHub Action to authenticate with Vault using an AppRole or token
2. **Secure Token Handling**: Ensure tokens are short-lived and have minimal permissions
3. **Automated Rotation**: Consider implementing automated secret rotation

Example GitHub Action using HashiCorp's Vault GitHub Action:

```yaml
name: Access Vault Secrets

jobs:
  example-job:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
        
      - name: Import Secrets
        uses: hashicorp/vault-action@v2
        with:
          url: ${{ secrets.VAULT_ADDR }}
          token: ${{ secrets.VAULT_TOKEN }}
          secrets: |
            secret/data/my-app/config API_KEY | API_KEY ;
            secret/data/my-app/db DB_PASSWORD | DATABASE_PASSWORD
            
      - name: Use Secrets
        run: |
          echo "Using secrets to perform tasks"
          # API_KEY and DATABASE_PASSWORD are now available as environment variables
```

## Environment-Specific Secrets

Different environments (development, staging, production) should have their own set of secrets. 

### Organizing Secrets by Environment

In Vault, organize secrets by environment path:

```
secret/
├── development/
│   ├── database
│   ├── api-keys
│   └── ...
├── staging/
│   ├── database
│   └── ...
└── production/
    ├── database
    └── ...
```

## Key Takeaways
- Vault centralizes and secures secrets, reducing risks associated with scattered credentials
- Properly setting up and managing Vault is crucial for security
- Use policies to define and refine access controls in Vault
- Ensure only trusted entities, whether they're VPS or Kubernetes workloads, have access to Vault
- In CI/CD, use platform-provided secret storage when possible
- For advanced use cases, integrate CI/CD pipelines with Vault
- Always separate secrets by environment to maintain proper isolation
- Regularly rotate secrets and audit access

## Troubleshooting

### Common Issues

1. **"Permission denied" errors**: Usually related to policy issues. Check that the token or entity has the necessary permissions.
2. **Sealed Vault**: If Vault becomes sealed (e.g., after a restart), you'll need to unseal it using unseal keys.
3. **Token expiration**: Tokens have TTLs. If a token expires, renew it or create a new one.
4. **Auth method configuration**: Make sure auth methods (like AppRole or Kubernetes) are correctly configured.

### Debugging Tools

- Use `vault debug` command for collecting runtime information
- Check Vault server logs for detailed error messages
- Use the Vault UI to verify policy configuration and secret existence 