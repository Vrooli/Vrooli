package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

func init() {
	registerManagedSharedBootstrapper("vault", func(ctx context.Context, host *UserResourceHost, instance ManagedInstance, appScope string) error {
		_, err := host.EnsureVault(ctx, instance, appScope, HTTPVaultBootstrapper{})
		return err
	})
}

// StartVaultBrokerControl exposes only scoped use/credential operations to
// installed applications. The owner token may authorize host management; app
// tokens cannot. The returned server contains no management material.
func (h *UserResourceHost) StartVaultBrokerControl(listener net.Listener, credentials map[string]string) (*BrokerControlServer, error) {
	if h == nil || h.Broker == nil {
		return nil, fmt.Errorf("user resource host broker is unavailable")
	}
	server, err := StartBrokerControlServer(listener, h.Broker, credentials)
	if err != nil {
		return nil, err
	}
	issuer := VaultCredentialIssuer{ManagementToken: h.VaultManagementToken}
	if err := server.RegisterCredentialIssuer("vault", issuer); err != nil {
		_ = server.Close(context.Background())
		return nil, err
	}
	return server, nil
}

// VaultBootstrapper performs Vault-native initialization and unseal work. It
// returns management material only to this resource-native boundary.
type VaultBootstrapper interface {
	Bootstrap(context.Context, string) (VaultBootstrapMaterial, error)
}

type VaultRecoveryBootstrapper interface {
	Unseal(context.Context, string, VaultBootstrapMaterial) error
}

type VaultBootstrapMaterial struct {
	RootToken string `json:"root_token"`
	UnsealKey string `json:"unseal_key"`
}

// EnsureVault stores Vault recovery material before the shared instance is
// leasable. It is recovery-safe: an initialized instance is unsealed using
// secure material, while a fresh instance initializes once.
func (h *UserResourceHost) EnsureVault(ctx context.Context, instance ManagedInstance, appScope string, bootstrap VaultBootstrapper) (ManagedInstance, error) {
	if h == nil || h.Broker == nil || h.Secrets == nil || bootstrap == nil {
		return ManagedInstance{}, fmt.Errorf("user resource host is incomplete")
	}
	if instance.Resource != "vault" || instance.Provider != resourcedeployment.ProviderManagedShared || instance.OwnerScope != h.OwnerScope || !isLoopbackManagedEndpoint(instance.Endpoint) {
		return ManagedInstance{}, fmt.Errorf("Vault user-host instance is not a verified owned loopback service")
	}
	if err := h.SecureStorageReady(instance.ID); err != nil {
		return ManagedInstance{}, fmt.Errorf("secure storage is not ready; refusing Vault initialization: %w", err)
	}
	var material VaultBootstrapMaterial
	if raw, err := h.Secrets.Get("vrooli.resource.vault", instance.ID); err == nil {
		if err := json.Unmarshal([]byte(raw), &material); err != nil {
			return ManagedInstance{}, fmt.Errorf("parse secure Vault management material: %w", err)
		}
		recovery, ok := bootstrap.(VaultRecoveryBootstrapper)
		if !ok {
			return ManagedInstance{}, fmt.Errorf("Vault bootstrap adapter cannot recover an initialized instance")
		}
		if err := recovery.Unseal(ctx, instance.Endpoint, material); err != nil {
			return ManagedInstance{}, fmt.Errorf("recover user-hosted Vault: %w", err)
		}
	} else {
		material, err = bootstrap.Bootstrap(ctx, instance.Endpoint)
		if err != nil {
			return ManagedInstance{}, fmt.Errorf("bootstrap user-hosted Vault: %w", err)
		}
	}
	if strings.TrimSpace(material.RootToken) == "" || strings.TrimSpace(material.UnsealKey) == "" {
		return ManagedInstance{}, fmt.Errorf("Vault bootstrap returned incomplete management material")
	}
	encoded, err := json.Marshal(material)
	if err != nil {
		return ManagedInstance{}, err
	}
	if err := h.Secrets.Put("vrooli.resource.vault", instance.ID, string(encoded)); err != nil {
		return ManagedInstance{}, fmt.Errorf("securely store Vault management material: %w", err)
	}
	registered, err := h.Broker.RegisterOrGrantScope(instance, appScope)
	if err != nil {
		return ManagedInstance{}, err
	}
	return registered, nil
}

// VaultManagementToken is the only bridge from secure storage to Vault's
// resource-native credential issuer. It never serializes the token into
// broker state or host status.
func (h *UserResourceHost) VaultManagementToken(instance ManagedInstance) (string, error) {
	if h == nil || h.Secrets == nil || instance.Resource != "vault" {
		return "", fmt.Errorf("Vault management token is unavailable")
	}
	raw, err := h.Secrets.Get("vrooli.resource.vault", instance.ID)
	if err != nil {
		return "", fmt.Errorf("read Vault management material: %w", err)
	}
	var material VaultBootstrapMaterial
	if err := json.Unmarshal([]byte(raw), &material); err != nil {
		return "", fmt.Errorf("parse secure Vault management material: %w", err)
	}
	if strings.TrimSpace(material.RootToken) == "" {
		return "", fmt.Errorf("secure Vault management material has no root token")
	}
	return material.RootToken, nil
}

// HTTPVaultBootstrapper implements Vault's documented local bootstrap API.
// All requests are loopback-only and bounded by the caller's context.
type HTTPVaultBootstrapper struct{ Client *http.Client }

func (b HTTPVaultBootstrapper) Bootstrap(ctx context.Context, endpoint string) (VaultBootstrapMaterial, error) {
	if !isLoopbackManagedEndpoint(endpoint) {
		return VaultBootstrapMaterial{}, fmt.Errorf("Vault bootstrap endpoint must be loopback")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	var status struct {
		Initialized bool `json:"initialized"`
		Sealed      bool `json:"sealed"`
	}
	if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err != nil {
		return VaultBootstrapMaterial{}, err
	}
	if !status.Initialized {
		var initialized struct {
			Keys      []string `json:"keys"`
			RootToken string   `json:"root_token"`
		}
		if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/init", map[string]int{"secret_shares": 1, "secret_threshold": 1}, &initialized); err != nil {
			return VaultBootstrapMaterial{}, err
		}
		if len(initialized.Keys) != 1 || strings.TrimSpace(initialized.RootToken) == "" {
			return VaultBootstrapMaterial{}, fmt.Errorf("Vault initialization returned incomplete material")
		}
		if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/unseal", map[string]string{"key": initialized.Keys[0]}, nil); err != nil {
			return VaultBootstrapMaterial{}, err
		}
		return VaultBootstrapMaterial{RootToken: initialized.RootToken, UnsealKey: initialized.Keys[0]}, nil
	}
	return VaultBootstrapMaterial{}, fmt.Errorf("initialized Vault recovery requires existing secure management material")
}

func (b HTTPVaultBootstrapper) Unseal(ctx context.Context, endpoint string, material VaultBootstrapMaterial) error {
	if !isLoopbackManagedEndpoint(endpoint) || strings.TrimSpace(material.UnsealKey) == "" {
		return fmt.Errorf("Vault recovery requires a loopback endpoint and secure unseal material")
	}
	client := b.Client
	if client == nil {
		client = &http.Client{}
	}
	var status struct {
		Sealed bool `json:"sealed"`
	}
	if err := vaultBootstrapRequest(ctx, client, endpoint, http.MethodGet, "/v1/sys/seal-status", nil, &status); err != nil {
		return err
	}
	if !status.Sealed {
		return nil
	}
	return vaultBootstrapRequest(ctx, client, endpoint, http.MethodPut, "/v1/sys/unseal", map[string]string{"key": material.UnsealKey}, nil)
}

func vaultBootstrapRequest(ctx context.Context, client *http.Client, endpoint, method, path string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Vault bootstrap API returned %s", response.Status)
	}
	if output == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(output)
}
