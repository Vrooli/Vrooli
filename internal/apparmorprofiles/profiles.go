// Package apparmorprofiles owns the fixed AppArmor profiles managed by setup.
package apparmorprofiles

type ID string

const (
	BubblewrapUserns       ID = "bubblewrap-userns"
	WorkspaceSandboxUserns ID = "workspace-sandbox-userns"
	// ProfileDirectory is the only directory managed by this package.
	ProfileDirectory = "/etc/apparmor.d"
	// BubblewrapProfileName is the profile loaded for distribution bubblewrap.
	BubblewrapProfileName = "vrooli-bubblewrap"
	// WorkspaceSandboxName is the profile loaded by workspace-sandbox.
	WorkspaceSandboxName = "vrooli-workspace-sandbox"
	// ParserReloadFlag reloads one fixed profile through apparmor_parser.
	ParserReloadFlag = "-r"
)

type Replacement uint8

const (
	RefuseDifferent Replacement = iota
	ReplaceManaged
)

type Profile struct {
	Name        string
	Path        string
	Content     string
	Replacement Replacement
	Parser      []string
	ParserArgs  []string
}

var parserCandidates = []string{
	"apparmor_parser",
	"/usr/sbin/apparmor_parser",
	"/sbin/apparmor_parser",
	"/usr/bin/apparmor_parser",
	"/bin/apparmor_parser",
}

func ParserCandidates() []string { return append([]string(nil), parserCandidates...) }

func Lookup(id string) (Profile, bool) {
	switch ID(id) {
	case BubblewrapUserns:
		return Profile{Name: BubblewrapProfileName, Path: ProfileDirectory + "/" + BubblewrapProfileName, Content: bubblewrap, Replacement: RefuseDifferent, Parser: ParserCandidates(), ParserArgs: []string{ParserReloadFlag}}, true
	case WorkspaceSandboxUserns:
		return Profile{Name: WorkspaceSandboxName, Path: ProfileDirectory + "/" + WorkspaceSandboxName, Content: workspaceSandbox, Replacement: ReplaceManaged, Parser: ParserCandidates(), ParserArgs: []string{ParserReloadFlag}}, true
	default:
		return Profile{}, false
	}
}

const bubblewrap = `# Managed by Vrooli -- do not edit manually
abi <abi/4.0>,
include <tunables/global>

# Opt in only the distribution bubblewrap executable. Do not disable the
# host's restricted-unprivileged-userns setting or broaden the service profile.
profile ` + BubblewrapProfileName + ` /{,usr/}bin/bwrap flags=(unconfined) {
  userns,
  include if exists <local/vrooli-bubblewrap>
}
`

const workspaceSandbox = `# Managed by Vrooli -- do not edit manually
abi <abi/4.0>,
include <tunables/global>

profile ` + WorkspaceSandboxName + ` flags=(unconfined) {
  # Ubuntu's restricted-unprivileged-userns policy only grants full
  # userns behavior to named profiles that explicitly opt in. The launcher
  # enters this profile for workspace-sandbox API startup only; this is
  # narrower than disabling AppArmor's global restriction.
  userns,

  include if exists <local/` + WorkspaceSandboxName + `>
}
`
