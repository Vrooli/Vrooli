# Secrets Management
It is important that secrets are managed properly. Using environment variables in Docker containers can lead to secrets being exposed when images are pushed to Docker Hub. So instead, we store secrets using HashiCorp Vault.

## Introduction to HashiCorp Vault

### What is HashiCorp Vault?
HashiCorp Vault is a tool designed to manage secrets and protect sensitive data. It provides a centralized location to store, access, and manage tokens, passwords, certificates, and encryption keys.

## Setting Up Vault

### Development Setup
Setting up Vault for development is straightforward. In development mode, Vault runs in-memory and is automatically initialized and unsealed.

```bash
vault server -dev
```

For detailed steps and scripts, refer to our detailed development setup guide (coming soon).

### Production Setup
Production setups require careful planning around storage backends, high availability, and secure access. Steps generally include:

1. Choosing a storage backend.
2. Configuring and starting the Vault server.
3. Initializing and unsealing Vault.

For a detailed production setup, refer to our comprehensive production setup guide (coming soon).

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

## Key Takeaways
- Vault centralizes and secures secrets, reducing risks associated with scattered credentials.
- Properly setting up and managing Vault is crucial for security.
- Use policies to define and refine access controls in Vault.
- Ensure only trusted entities, whether they're VPS or Kubernetes workloads, have access to Vault.

