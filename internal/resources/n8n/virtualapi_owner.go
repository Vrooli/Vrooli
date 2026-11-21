package n8n

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/resources/n8n/virtualapi"
)

const (
	virtualAPIContainerResource = "n8n-virtual-api"
	virtualAPIContainerKind     = "virtual-api"
	virtualAPIPort              = 443
	virtualAPITmpfsBytes        = "32m"
	maxVirtualAPIHitsBytes      = 1 << 20
)

const virtualAPIHitsReaderProgram = `const fs=require("node:fs");process.stdout.write(fs.readFileSync("/tmp/virtualapi-hits.json"));`

// VirtualAPIReceipt is content-free evidence that a task-private HTTPS
// companion is attached to n8n's internal network and answers only its
// declared aliases/routes.
type VirtualAPIReceipt struct {
	SandboxID          string   `json:"sandboxId"`
	CompanionRuntimeID string   `json:"companionRuntimeId"`
	Aliases            []string `json:"aliases"`
	Port               int      `json:"port"`
	ProfileFingerprint string   `json:"profileFingerprintSha256"`
	CAFingerprint      string   `json:"caFingerprintSha256"`
	RouteCount         int      `json:"routeCount"`
	// NoEgressVerified is set only after both owned containers are verified on
	// the task's Docker Internal network, with no other network attachments or
	// published API-companion ports. The fixed server has no dial or proxy path;
	// workflow route hits separately prove the declared aliases.
	NoEgressVerified bool   `json:"noEgressVerified"`
	Status           string `json:"status"`
}

type VirtualAPIRouteHit struct {
	Host        string `json:"host"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	QuerySHA256 string `json:"queryHash"`
	Count       uint64 `json:"count"`
}

type VirtualAPIHitsReceipt struct {
	SandboxID          string               `json:"sandboxId"`
	CompanionRuntimeID string               `json:"companionRuntimeId"`
	Status             string               `json:"status"`
	TotalHits          uint64               `json:"totalHits"`
	Routes             []VirtualAPIRouteHit `json:"routes"`
}

// StartVirtualAPIContract is the supported byte-boundary wrapper used by the
// resource CLI and Workspace Sandbox provider. The gateway key material stays
// in this process and the container receives only the leaf server key.
func (o *Owner) StartVirtualAPIContract(ctx context.Context, sandboxID string, contractJSON []byte) (VirtualAPIReceipt, error) {
	if len(contractJSON) == 0 || len(contractJSON) > 2<<20 {
		return VirtualAPIReceipt{}, errors.New("virtual API contract must contain 1..2 MiB")
	}
	uniqueDecoder := json.NewDecoder(strings.NewReader(string(contractJSON)))
	if err := checkUniqueJSONValue(uniqueDecoder); err != nil {
		return VirtualAPIReceipt{}, fmt.Errorf("decode virtual API contract: %w", err)
	}
	if _, err := uniqueDecoder.Token(); err != io.EOF {
		return VirtualAPIReceipt{}, errors.New("virtual API contract contains trailing data")
	}
	decoder := json.NewDecoder(strings.NewReader(string(contractJSON)))
	decoder.DisallowUnknownFields()
	var contract virtualapi.Contract
	if err := decoder.Decode(&contract); err != nil {
		return VirtualAPIReceipt{}, fmt.Errorf("decode virtual API contract: %w", err)
	}
	gateway, receipt, err := virtualapi.New(contract)
	if err != nil {
		return VirtualAPIReceipt{}, err
	}
	defer gateway.Close()
	return o.startVirtualAPI(ctx, sandboxID, gateway, receipt)
}

// StartVirtualAPI starts n8n with the gateway CA trusted before Node boots,
// then launches the fixed HTTPS adapter in a fresh, non-egress container. The
// adapter begins on network=none and is attached only to this task's internal
// network with the declared real API hostnames.
func (o *Owner) startVirtualAPI(ctx context.Context, sandboxID string, gateway *virtualapi.Gateway, gatewayReceipt virtualapi.Receipt) (result VirtualAPIReceipt, resultErr error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return VirtualAPIReceipt{}, err
	}
	if gateway == nil || gatewayReceipt.TaskID != canonical || gatewayReceipt.Port != virtualAPIPort || gatewayReceipt.NoOutboundDial != true || gatewayReceipt.NoEgressVerified {
		return VirtualAPIReceipt{}, errors.New("virtual API task identity or initial safety receipt is invalid")
	}
	if len(gatewayReceipt.Aliases) == 0 || len(gatewayReceipt.Aliases) > virtualapi.MaxHosts || !sameStrings(gatewayReceipt.Aliases, sortedUniqueHosts(gatewayReceipt.Aliases)) {
		return VirtualAPIReceipt{}, errors.New("virtual API aliases are invalid or not canonical")
	}
	caHash := sha256.Sum256(gatewayReceipt.CACertPEM)
	if hex.EncodeToString(caHash[:]) != gatewayReceipt.CAFingerprint {
		return VirtualAPIReceipt{}, errors.New("virtual API trust CA fingerprint does not match its certificate")
	}
	configJSON, err := gateway.ServerConfigJSON()
	if err != nil {
		return VirtualAPIReceipt{}, fmt.Errorf("prepare virtual API configuration: %w", err)
	}
	if len(configJSON) == 0 || len(configJSON) > virtualapi.MaxServerConfigBytes {
		return VirtualAPIReceipt{}, errors.New("virtual API configuration exceeds its bound")
	}
	var config virtualapi.ContainerServerConfig
	if err := json.Unmarshal(configJSON, &config); err != nil || len(config.Routes) == 0 || len(config.Routes) > virtualapi.MaxRoutes {
		return VirtualAPIReceipt{}, errors.New("virtual API configuration has an invalid route profile")
	}
	configHash := gatewayReceipt.ProfileFingerprint
	if !validSHA256(configHash) {
		return VirtualAPIReceipt{}, errors.New("virtual API profile fingerprint is invalid")
	}
	if o.image() != ImageRef {
		return VirtualAPIReceipt{}, errors.New("virtual API companion must use the pinned n8n image")
	}
	if ok, err := o.imagePresent(ctx); err != nil {
		return VirtualAPIReceipt{}, err
	} else if !ok {
		return VirtualAPIReceipt{}, ErrImageMissing
	}
	before, err := o.Status(ctx, canonical)
	if err != nil {
		return VirtualAPIReceipt{}, err
	}
	createdRuntime := before.Status == "stopped"
	completed := false
	defer func() {
		if !createdRuntime || completed {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, cleanupErr := o.Stop(cleanupCtx, canonical, runtimeIDFor(canonical))
		if cleanupErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("clean task runtime after virtual API start failure: %w", cleanupErr))
		}
	}()
	if _, err := o.StartWithTaskTrust(ctx, canonical, gatewayReceipt.CACertPEM); err != nil {
		return VirtualAPIReceipt{}, fmt.Errorf("start task runtime with virtual API trust: %w", err)
	}
	runtimeID := virtualAPIRuntimeIDFor(canonical, configHash)
	containerName := virtualAPIContainerName(canonical)
	_, taskNetwork := objectNames(canonical)
	if existing, found, err := o.inspectContainer(ctx, containerName); err != nil {
		return VirtualAPIReceipt{}, err
	} else if found {
		if err := verifyVirtualAPIIdentity(existing, canonical, runtimeID, configHash, gatewayReceipt.Aliases); err != nil {
			return VirtualAPIReceipt{}, err
		}
		if err := o.verifyVirtualAPINetworks(ctx, canonical, existing, gatewayReceipt.Aliases); err != nil {
			return VirtualAPIReceipt{}, err
		}
		if err := o.ensureVirtualAPIAliasesExclusive(ctx, canonical, gatewayReceipt.Aliases, existing.ID); err != nil {
			return VirtualAPIReceipt{}, err
		}
		if !existing.State.Running {
			return VirtualAPIReceipt{}, errors.New("existing virtual API companion is stopped; stop and recreate it with its task runtime")
		}
		if err := o.verifyVirtualAPITaskIsolation(ctx, canonical, existing, gatewayReceipt.Aliases); err != nil {
			return VirtualAPIReceipt{}, err
		}
		completed = true
		return makeVirtualAPIReceipt(canonical, runtimeID, gatewayReceipt, len(config.Routes)), nil
	}
	if _, found, err := o.inspectNetwork(ctx, taskNetwork, canonical, runtimeIDFor(canonical)); err != nil || !found {
		if err == nil {
			err = errors.New("task-owned n8n internal network is missing")
		}
		return VirtualAPIReceipt{}, err
	}
	if err := o.ensureVirtualAPIAliasesExclusive(ctx, canonical, gatewayReceipt.Aliases, ""); err != nil {
		return VirtualAPIReceipt{}, err
	}
	if err := o.ensureVirtualAPIIsolatedNetwork(ctx, canonical); err != nil {
		return VirtualAPIReceipt{}, err
	}
	args := []string{
		"run", "--detach", "--name", containerName, "--network", "none",
		"--read-only", "--user", "node", "--cap-drop", "ALL", "--cap-add", "NET_BIND_SERVICE",
		"--security-opt", "no-new-privileges:true", "--pids-limit", "64", "--memory", "256m", "--memory-swap", "256m", "--cpus", "1",
		"--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=" + virtualAPITmpfsBytes + ",uid=1000,gid=1000,mode=0700",
		"--label", labelResource + "=n8n", "--label", labelSandboxID + "=" + canonical,
		"--label", labelRuntimeID + "=" + runtimeID, "--label", labelObjectKind + "=" + virtualAPIContainerKind,
		"--label", labelCompanionResource + "=" + virtualAPIContainerResource,
		"--label", labelSandboxOwner + "=" + canonical, "--label", labelSandboxRunID + "=" + runtimeID,
		"--label", labelSandboxDecl + "=true", "--label", labelHTTPMockConfigHash + "=" + configHash,
		"--label", labelHTTPMockAliases + "=" + strings.Join(gatewayReceipt.Aliases, ","),
		"--entrypoint", "/bin/sh", o.image(), "-c",
		"while [ ! -s /tmp/virtualapi-start ]; do sleep 0.05; done; exec node --max-old-space-size=64 /tmp/virtualapi-server.js /tmp/virtualapi-config.json",
	}
	createdID, err := o.runner().Run(ctx, args...)
	if err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(fmt.Errorf("create isolated virtual API companion: %w", err), cleanupErr)
	}
	containerID := strings.TrimSpace(string(createdID))
	if !companionRuntimePattern.MatchString(containerID) {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(errors.New("Docker returned an invalid virtual API container ID"), cleanupErr)
	}
	container, found, err := o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		if err == nil {
			err = errors.New("started virtual API container cannot be inspected")
		}
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := verifyVirtualAPIIdentity(container, canonical, runtimeID, configHash, gatewayReceipt.Aliases); err != nil {
		return VirtualAPIReceipt{}, err
	}
	quiet, ok := o.runner().(quietCommandRunner)
	if !ok {
		return VirtualAPIReceipt{}, errors.New("Docker runner cannot stage bounded virtual API service data")
	}
	serverProgram := []byte(virtualapi.ContainerServerProgram)
	for _, file := range virtualAPIStagedFiles(configJSON, gatewayReceipt.CACertPEM, serverProgram) {
		if _, err := quiet.RunInputQuiet(ctx, file.payload, "exec", "--interactive", "--user", "node", containerID, "node", "-e", stageWorkflowFileProgram, file.path); err != nil {
			cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
			return VirtualAPIReceipt{}, errors.Join(fmt.Errorf("stage bounded virtual API task file: %w", err), cleanupErr)
		}
	}
	if err := o.waitVirtualAPIReady(ctx, containerID, gatewayReceipt.Aliases[0]); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	connectArgs := []string{"network", "connect"}
	for _, alias := range gatewayReceipt.Aliases {
		connectArgs = append(connectArgs, "--alias", alias)
	}
	connectArgs = append(connectArgs, taskNetwork, containerID)
	// Docker treats network=none as a private network that cannot coexist
	// with any other network attachment. The container starts offline; remove
	// that special attachment before connecting the internal-only task network.
	if _, err := o.runner().Run(ctx, "network", "disconnect", "none", containerID); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(fmt.Errorf("detach virtual API companion from network=none: %w", err), cleanupErr)
	}
	if err := o.ensureVirtualAPIAliasesExclusive(ctx, canonical, gatewayReceipt.Aliases, containerID); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	if _, err := o.runner().Run(ctx, connectArgs...); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(fmt.Errorf("attach virtual API aliases to task network: %w", err), cleanupErr)
	}
	container, found, err = o.inspectContainer(ctx, containerName)
	if err != nil || !found {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		if err == nil {
			err = errors.New("virtual API container disappeared before receipt verification")
		}
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := o.verifyVirtualAPINetworks(ctx, canonical, container, gatewayReceipt.Aliases); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	if err := o.verifyVirtualAPITaskIsolation(ctx, canonical, container, gatewayReceipt.Aliases); err != nil {
		cleanupErr := o.cleanupVirtualAPIByName(context.Background(), canonical, runtimeID, configHash, containerName)
		return VirtualAPIReceipt{}, errors.Join(err, cleanupErr)
	}
	completed = true
	return makeVirtualAPIReceipt(canonical, runtimeID, gatewayReceipt, len(config.Routes)), nil
}

type virtualAPIStagedFile struct {
	path    string
	payload []byte
}

// The companion's entrypoint starts the server as soon as its configuration
// and program are both present. Stage the CA first and the server program last
// so staging never launches a second Node process beside the resident server.
func virtualAPIStagedFiles(configJSON, caPEM, serverProgram []byte) []virtualAPIStagedFile {
	return []virtualAPIStagedFile{
		{path: "/tmp/virtualapi-config.json", payload: configJSON},
		{path: "/tmp/virtualapi-ca.pem", payload: caPEM},
		{path: "/tmp/virtualapi-server.js", payload: serverProgram},
		// A separate marker makes readiness independent of a server file that
		// becomes nonempty while Docker is still streaming its bytes.
		{path: "/tmp/virtualapi-start", payload: []byte("start\n")},
	}
}

func (o *Owner) ensureVirtualAPIAliasesExclusive(ctx context.Context, sandboxID string, aliases []string, selfID string) error {
	_, taskNetwork := objectNames(sandboxID)
	network, found, err := o.inspectNetwork(ctx, taskNetwork, sandboxID, runtimeIDFor(sandboxID))
	if err != nil {
		return err
	}
	if !found || !network.Internal {
		return errors.New("virtual API hostname checks require the task internal network")
	}
	wanted := map[string]bool{}
	for _, alias := range aliases {
		wanted[alias] = true
	}
	for id := range network.Containers {
		if id == selfID {
			continue
		}
		member, err := o.inspectContainerByID(ctx, id)
		if err != nil {
			return fmt.Errorf("inspect task network member before API alias attach: %w", err)
		}
		state, attached := member.NetworkSettings.Networks[taskNetwork]
		if !attached {
			return errors.New("task network membership changed while verifying virtual API aliases")
		}
		for _, actual := range state.Aliases {
			if wanted[actual] {
				return fmt.Errorf("virtual API hostname %q is already claimed on the task network", actual)
			}
		}
	}
	return nil
}

func (o *Owner) ensureVirtualAPIIsolatedNetwork(ctx context.Context, sandboxID string) error {
	_, taskNetwork := objectNames(sandboxID)
	network, found, err := o.inspectNetwork(ctx, taskNetwork, sandboxID, runtimeIDFor(sandboxID))
	if err != nil {
		return err
	}
	if !found || !network.Internal {
		return errors.New("virtual API requires the task-owned internal-only network")
	}
	return nil
}

func (o *Owner) verifyVirtualAPINetworks(ctx context.Context, sandboxID string, container containerInspect, aliases []string) error {
	_, taskNetwork := objectNames(sandboxID)
	if err := verifyNoEgressExceptTask(container, taskNetwork); err != nil {
		return err
	}
	if err := o.ensureVirtualAPIIsolatedNetwork(ctx, sandboxID); err != nil {
		return err
	}
	state, attached := container.NetworkSettings.Networks[taskNetwork]
	if !attached || !validActualHTTPMockAliases(aliases, state.Aliases, virtualAPIContainerName(sandboxID)) {
		return errors.New("virtual API aliases do not match task network attachment")
	}
	for _, bindings := range container.NetworkSettings.Ports {
		if len(bindings) > 0 {
			return errors.New("virtual API unexpectedly publishes a host port")
		}
	}
	return nil
}

func (o *Owner) waitVirtualAPIReady(ctx context.Context, containerID, serverName string) error {
	quiet, ok := o.runner().(quietCommandRunner)
	if !ok {
		return errors.New("Docker runner cannot guarantee bounded virtual API probe output")
	}
	probe := fmt.Sprintf(`const fs=require("node:fs"),tls=require("node:tls");let done=false;function finish(v){if(done)return;done=true;process.stdout.write(v);process.exit(0)}const s=tls.connect({host:"127.0.0.1",port:443,servername:%q,ca:fs.readFileSync("/tmp/virtualapi-ca.pem"),rejectUnauthorized:true,timeout:1500},()=>{const ok=s.authorized===true;s.end();finish(ok?"ready":"certificate-untrusted")});s.on("timeout",()=>{s.destroy();finish("timeout")});s.on("error",e=>{const allowed=new Set(["ECONNREFUSED","ECONNRESET","ETIMEDOUT","CERT_HAS_EXPIRED","DEPTH_ZERO_SELF_SIGNED_CERT","ERR_TLS_CERT_ALTNAME_INVALID","UNABLE_TO_VERIFY_LEAF_SIGNATURE"]);finish(allowed.has(e.code)?e.code:"tls-error")});`, serverName)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	lastResult := "no-result"
	for attempt := 0; attempt < 30; attempt++ {
		output, exceeded, err := quiet.RunStdoutLimitQuiet(ctx, 32, "exec", "--user", "node", containerID, "node", "--max-old-space-size=32", "-e", probe)
		if err == nil && exceeded {
			lastResult = "probe-output-exceeded"
		} else if err != nil {
			lastResult = "probe-command-failed"
			if strings.Contains(err.Error(), "exit status 137") {
				lastResult = "probe-process-killed"
			}
			if state, found, inspectErr := o.inspectContainer(ctx, containerID); inspectErr == nil && found && !state.State.Running {
				if state.State.OOMKilled {
					lastResult = "companion-oom-killed"
				} else {
					lastResult = fmt.Sprintf("companion-exited-%d-%s", boundedContainerExitCode(state.State.ExitCode), o.classifyVirtualAPIStartupLogs(ctx, quiet, containerID))
				}
			}
		} else {
			lastResult = string(output)
		}
		if err == nil && !exceeded && lastResult == "ready" {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("virtual API HTTPS listener did not become ready (probe=%s)", lastResult)
		case <-time.After(250 * time.Millisecond):
		}
	}
	return fmt.Errorf("virtual API HTTPS listener failed its trusted-certificate readiness probe (probe=%s)", lastResult)
}

func boundedContainerExitCode(exitCode int) int {
	if exitCode < 0 || exitCode > 255 {
		return -1
	}
	return exitCode
}

// classifyVirtualAPIStartupLogs maps a small fixed set of server startup
// failures to content-free classes. Raw logs can contain fixture data, so they
// never cross the owner boundary.
func (o *Owner) classifyVirtualAPIStartupLogs(ctx context.Context, quiet quietCommandRunner, containerID string) string {
	logs, exceeded, err := quiet.RunStdoutLimitQuiet(ctx, 8<<10, "logs", "--tail", "20", containerID)
	if err != nil || exceeded {
		return "startup-log-unavailable"
	}
	text := strings.ToLower(string(logs))
	switch {
	case strings.Contains(text, "heap out of memory"), strings.Contains(text, "allocation failed"):
		return "server-memory-exhausted"
	case strings.Contains(text, "eacces"):
		return "server-bind-permission-denied"
	case strings.Contains(text, "eaddrinuse"):
		return "server-bind-address-conflict"
	case strings.Contains(text, "invalid gateway config"), strings.Contains(text, "invalid exact hostname"), strings.Contains(text, "invalid exact fixture"):
		return "server-rejected-config"
	case strings.Contains(text, "syntaxerror"):
		return "server-program-syntax-error"
	case strings.Contains(text, "typeerror"):
		return "server-program-type-error"
	case len(logs) == 0:
		return "no-startup-log"
	default:
		return "other-startup-error"
	}
}

func (o *Owner) verifyVirtualAPITaskIsolation(ctx context.Context, sandboxID string, apiContainer containerInspect, aliases []string) error {
	_, taskNetwork := objectNames(sandboxID)
	if err := o.verifyVirtualAPINetworks(ctx, sandboxID, apiContainer, aliases); err != nil {
		return err
	}
	runtimeContainer, found, err := o.inspectContainer(ctx, objectNamesValue(sandboxID))
	if err != nil || !found {
		if err == nil {
			err = errors.New("task-owned n8n runtime disappeared")
		}
		return err
	}
	if err := verifyN8nLabels(runtimeContainer.Config.Labels, sandboxID, runtimeIDFor(sandboxID), "container"); err != nil {
		return err
	}
	if runtimeContainer.Config.Labels[labelTaskCAFingerprint] == "" {
		return errors.New("n8n runtime has no task trust CA configured")
	}
	if err := verifyNoEgressExceptTask(runtimeContainer, taskNetwork); err != nil {
		return err
	}
	return nil
}

func (o *Owner) StopVirtualAPI(ctx context.Context, sandboxID, runtimeID string) (VirtualAPIReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return VirtualAPIReceipt{}, err
	}
	name := virtualAPIContainerName(canonical)
	container, found, err := o.inspectContainer(ctx, name)
	if err != nil || !found {
		return VirtualAPIReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, Status: "stopped"}, err
	}
	profileFingerprint := container.Config.Labels[labelHTTPMockConfigHash]
	if !validSHA256(profileFingerprint) || runtimeID != virtualAPIRuntimeIDFor(canonical, profileFingerprint) {
		return VirtualAPIReceipt{}, ErrForeignObject
	}
	if err := verifyVirtualAPIIdentity(container, canonical, runtimeID, profileFingerprint, parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])); err != nil {
		return VirtualAPIReceipt{}, err
	}
	if err := o.verifyVirtualAPINetworks(ctx, canonical, container, parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])); err != nil {
		if orphanErr := verifyOrphanedHTTPMock(container); orphanErr != nil {
			return VirtualAPIReceipt{}, err
		}
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "5", container.ID); err != nil {
			return VirtualAPIReceipt{}, fmt.Errorf("stop task virtual API companion: %w", err)
		}
	}
	if _, err := o.runner().Run(ctx, "rm", "--volumes", container.ID); err != nil && !isNotFound(err) {
		return VirtualAPIReceipt{}, fmt.Errorf("remove task virtual API companion: %w", err)
	}
	return VirtualAPIReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, Status: "stopped"}, nil
}

// ReadVirtualAPIHits returns the private companion's content-free counters.
// It never reads or exposes request headers or bodies.
func (o *Owner) ReadVirtualAPIHits(ctx context.Context, sandboxID, runtimeID string) (VirtualAPIHitsReceipt, error) {
	canonical, err := normalizeSandboxID(sandboxID)
	if err != nil {
		return VirtualAPIHitsReceipt{}, err
	}
	container, found, err := o.inspectContainer(ctx, virtualAPIContainerName(canonical))
	if err != nil {
		return VirtualAPIHitsReceipt{}, err
	}
	if !found || !container.State.Running {
		return VirtualAPIHitsReceipt{}, errors.New("owned virtual API companion is not running")
	}
	profileFingerprint := container.Config.Labels[labelHTTPMockConfigHash]
	aliases := parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])
	if !validSHA256(profileFingerprint) || runtimeID != virtualAPIRuntimeIDFor(canonical, profileFingerprint) {
		return VirtualAPIHitsReceipt{}, ErrForeignObject
	}
	if err := verifyVirtualAPIIdentity(container, canonical, runtimeID, profileFingerprint, aliases); err != nil {
		return VirtualAPIHitsReceipt{}, err
	}
	if err := o.verifyVirtualAPINetworks(ctx, canonical, container, aliases); err != nil {
		return VirtualAPIHitsReceipt{}, err
	}
	quiet, ok := o.runner().(quietCommandRunner)
	if !ok {
		return VirtualAPIHitsReceipt{}, errors.New("Docker runner cannot guarantee bounded virtual API receipt output")
	}
	output, exceeded, err := quiet.RunStdoutLimitQuiet(ctx, maxVirtualAPIHitsBytes, "exec", "--user", "node", container.ID, "node", "-e", virtualAPIHitsReaderProgram)
	if err != nil || exceeded {
		return VirtualAPIHitsReceipt{}, errors.New("read virtual API route counters failed or exceeded the receipt bound")
	}
	var stored struct {
		Version int                  `json:"version"`
		Total   uint64               `json:"total"`
		Routes  []VirtualAPIRouteHit `json:"routes"`
	}
	uniqueDecoder := json.NewDecoder(strings.NewReader(string(output)))
	if err := checkUniqueJSONValue(uniqueDecoder); err != nil {
		return VirtualAPIHitsReceipt{}, errors.New("virtual API route-counter receipt is invalid")
	}
	if _, err := uniqueDecoder.Token(); err != io.EOF {
		return VirtualAPIHitsReceipt{}, errors.New("virtual API route-counter receipt contains trailing data")
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&stored); err != nil || stored.Version != 1 || len(stored.Routes) == 0 || len(stored.Routes) > virtualapi.MaxRoutes {
		return VirtualAPIHitsReceipt{}, errors.New("virtual API route-counter receipt is invalid")
	}
	allowedHosts := map[string]bool{}
	for _, host := range aliases {
		allowedHosts[host] = true
	}
	seen := map[string]bool{}
	var total uint64
	for _, route := range stored.Routes {
		key := route.Method + " " + route.Host + route.Path + "?" + route.QuerySHA256
		if !allowedHosts[route.Host] || !validVirtualAPIMethod(route.Method) || route.Path == "" || route.Path[0] != '/' || !validSHA256(route.QuerySHA256) || seen[key] || ^uint64(0)-total < route.Count {
			return VirtualAPIHitsReceipt{}, errors.New("virtual API route-counter identity or total is invalid")
		}
		seen[key] = true
		total += route.Count
	}
	if total != stored.Total {
		return VirtualAPIHitsReceipt{}, errors.New("virtual API route-counter total does not match its route counts")
	}
	sort.Slice(stored.Routes, func(i, j int) bool {
		a, b := stored.Routes[i], stored.Routes[j]
		if a.Host != b.Host {
			return a.Host < b.Host
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.QuerySHA256 < b.QuerySHA256
	})
	return VirtualAPIHitsReceipt{SandboxID: canonical, CompanionRuntimeID: runtimeID, Status: "running", TotalHits: total, Routes: stored.Routes}, nil
}

func validVirtualAPIMethod(method string) bool {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return true
	default:
		return false
	}
}

func (o *Owner) cleanupVirtualAPIByName(ctx context.Context, sandboxID, runtimeID, configHash, name string) error {
	container, found, err := o.inspectContainer(ctx, name)
	if err != nil || !found {
		return err
	}
	if err := verifyVirtualAPIIdentity(container, sandboxID, runtimeID, configHash, parseHTTPMockAliases(container.Config.Labels[labelHTTPMockAliases])); err != nil {
		return err
	}
	if container.State.Running {
		if _, err := o.runner().Run(ctx, "stop", "--time", "3", container.ID); err != nil {
			return err
		}
	}
	_, err = o.runner().Run(ctx, "rm", "--volumes", container.ID)
	if err != nil && !isNotFound(err) {
		return err
	}
	return nil
}

func verifyVirtualAPIIdentity(container containerInspect, sandboxID, runtimeID, configHash string, aliases []string) error {
	labels := container.Config.Labels
	if labels[labelSandboxID] != sandboxID || labels[labelRuntimeID] != runtimeID || labels[labelSandboxOwner] != sandboxID || labels[labelSandboxRunID] != runtimeID || labels[labelSandboxDecl] != "true" || labels[labelCompanionResource] != virtualAPIContainerResource || labels[labelObjectKind] != virtualAPIContainerKind || labels[labelHTTPMockConfigHash] != configHash || !sameStrings(aliases, sortedUniqueHosts(aliases)) {
		return ErrForeignObject
	}
	return nil
}

func makeVirtualAPIReceipt(sandboxID, runtimeID string, gatewayReceipt virtualapi.Receipt, routeCount int) VirtualAPIReceipt {
	return VirtualAPIReceipt{SandboxID: sandboxID, CompanionRuntimeID: runtimeID, Aliases: append([]string(nil), gatewayReceipt.Aliases...), Port: gatewayReceipt.Port, ProfileFingerprint: gatewayReceipt.ProfileFingerprint, CAFingerprint: gatewayReceipt.CAFingerprint, RouteCount: routeCount, NoEgressVerified: true, Status: "attached"}
}

func virtualAPIRuntimeIDFor(sandboxID, profileFingerprint string) string {
	digest := sha256.Sum256([]byte("vrooli-n8n-virtualapi-v1\x00" + sandboxID + "\x00" + profileFingerprint))
	return "virtualapi:" + hex.EncodeToString(digest[:12])
}

func virtualAPIContainerName(sandboxID string) string {
	return "vrooli-n8n-virtualapi-" + strings.ReplaceAll(sandboxID, "-", "")
}

func sortedUniqueHosts(hosts []string) []string {
	out := append([]string(nil), hosts...)
	sort.Strings(out)
	unique := out[:0]
	for _, host := range out {
		if net.ParseIP(host) != nil || strings.ContainsAny(host, "/:@*%\\") || host == "" || strings.TrimSpace(host) != host {
			return nil
		}
		if len(unique) == 0 || unique[len(unique)-1] != host {
			unique = append(unique, host)
		}
	}
	return unique
}

func validSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}

func objectNamesValue(sandboxID string) string {
	containerName, _ := objectNames(sandboxID)
	return containerName
}
