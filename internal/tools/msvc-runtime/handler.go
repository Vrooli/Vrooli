package msvcruntime

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type handler struct{ manifest hostreqkit.ToolManifest }

func NewHandler(m hostreqkit.ToolManifest) hostreqkit.Handler { return handler{m} }
func (h handler) Name() string                                { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind                      { return hostreqspec.KindTool }
func (h handler) Inspect(host hostreqkit.Host, r hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	s := hostreqkit.BaseStatus(r)
	s.Notes = append(s.Notes, h.manifest.InstallHint)
	if host.OS != "windows" {
		s.SupportClass = hostreqkit.SupportUnsupported
		s.ExecutionState = hostreqkit.ExecutionUnsupported
		return s
	}
	s.SupportClass = hostreqkit.SupportSupported
	s.PackageName = h.manifest.Packages["winget"]
	s.InstallSupported = host.PackageManager == "winget"
	version, err := Check()
	s.Installed = err == nil
	s.Version = version
	if err != nil {
		s.Notes = append(s.Notes, err.Error())
	} else {
		s.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	}
	return s
}

func (h handler) Apply(host hostreqkit.Host, s hostreqkit.ItemStatus, o hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	if s, done := hostreqkit.ApplyStatus(s); done {
		return s, nil
	}
	if !s.InstallSupported || host.OS != "windows" || host.PackageManager != "winget" {
		return s, fmt.Errorf("MSVC runtime requires supported native Windows winget setup")
	}
	command, args, err := hostreqkit.InstallCommand(host, h.manifest.Packages["winget"], o.SudoMode)
	if err != nil {
		return s, err
	}
	args = append(args, "--exact", "--source", "winget")
	if err := hostreqkit.RunInstallCommand(command, args, o); err != nil {
		return s, err
	}
	if o.DryRun {
		return s, nil
	}
	version, err := Check()
	s.Version = version
	s.Installed = err == nil
	if err != nil {
		return s, err
	}
	return s, nil
}
