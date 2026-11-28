# Mac signature artifact facts and caller boundary

This bounded followup inspected the pinned official archives without executing
their code or any macOS validation API. It does not establish native signature,
resource, AMFI, Gatekeeper, certificate-trust or execution qualification.

## Exact artifacts

| Fact | Intel | Apple Silicon |
| --- | --- | --- |
| Original archive SHA256 | `5b30d3c97bced335a40593319cb4c045f1ad818a0e0ddefd92350a695a7a6fc4` | `b370d3365eaae8b6a9217e7d1b1ffd5fb6ddffaaff3395fd3cac3673df83191f` |
| Main executable SHA256 | `d8ff22e19f7ee320c43bb62373124a025a4511be1e8de5868389c7b9a92040a0` | `f06aeb3e974576f0b403d0db849e176c048ac0853ec4a3d3f7d13672c479f611` |
| Main LC_CODE_SIGNATURE | Absent | Offset10981408, size39824 |
| Top Contents/_CodeSignature/CodeResources | Absent | 25351 bytes, SHA256 `f260e0a67ae634ec2da234b03b77803762e4eb96fb311c98e648d3462142bf8b` |
| Derived 77-image tree SHA256 | `61ead4311d86af4c89fdbfd2868b474dae3a2242216b6a506bd295336f2e2519` | `b777a233d74db1250f2a100571bb3880fa8975cc6eca6ee71fc44efe3f2cdab5` |

The Apple Silicon executable contains a CodeDirectory, requirements blob and
CMS signature blob. The CodeDirectory identifies `com.yourcompany.OpenRGB`,
has flags0, SHA256 hashes and 671 code slots at 16384-byte pages. All 671 page
hashes matched the pinned executable offline. Its special resource slot3
matches the exact original CodeResources SHA256 above. Blob presence and hash
agreement do not verify the CMS signature, signer trust, certificate validity
or native OS policy.

The version2 `files2` resource table contains a nested-code record for
`Frameworks/QtCore5Compat.framework`, with cdhash
`67d591fd65ae393418a1ca42f53fa76f36302e92` and a matching cdhash requirement.
It has no `optional` flag. The framework rule marks code under Frameworks as
nested; no applicable omission rule excludes this framework. The derived tree
removes it while preserving the original CodeResources and executable bytes.

## What can and cannot validate

Apple's [code-signing guidance](https://developer.apple.com/library/archive/technotes/tn2206/)
describes nested code as part of the outer resource seal and requires signed
bundles to remain unchanged. Its [static validation flags](https://developer.apple.com/documentation/security/static-code-validation-flags)
distinguish executable validation from resource validation. Public
[resource-seal parsing](https://github.com/apple-oss-distributions/Security/blob/main/OSX/libsecurity_codesigning/lib/resources.cpp)
defaults optionality to false; [static validation](https://github.com/apple-oss-distributions/Security/blob/main/OSX/libsecurity_codesigning/lib/StaticCode.cpp)
checks resources unless explicitly excluded and reports missing required
sealed resources as a failure.

Consequently, the derived Apple Silicon app cannot preserve a valid original
whole-bundle resource seal with this required nested framework absent. That
is an artifact-based inference, not an observed `codesign`, Security.framework
or Gatekeeper result; another earlier signature/trust error could also occur.
Intel has no signed top-level main executable/resource envelope to preserve.
Neither profile is qualified as a valid signed app bundle by this evidence.

## Real maintained caller

The resource opens the retained source and verifies the complete derived tree,
recipe metadata, exact stage layout, native ABI and loader graph in
`internal/resources/openrgb_appimage_stage.go`. Native preparation in
`openrgb_native_capsule_child.go` additionally checks system prerequisites and
opens the exact payload. Darwin handoff in
`packages/platform-go/process_image_darwin.go` and `darwin_image_exec.go`
checks machine-owned immutable ancestry, the held file/path vnode, executable
kind/CPU and explicit environment, then calls `syscall.Exec` on that pathname.

This caller does not invoke `SecStaticCodeCheckValidity`, `codesign`, `spctl`
or LaunchServices. It does not set flags to skip native resource validation,
force-remove quarantine, or retry through a signature-bypass path. Its object
and dependency checks are project admission checks, **not** native signature
validation. Direct exec does not prove that macOS will accept the code or that
native policy will ignore the surrounding app resource seal. Kernel/AMFI/dyld
and any applicable assessment policy remain authoritative and unobserved here.

The next real target qualification must record the exact deployed derived
tree, native object and full-bundle validation results separately, then the
actual ordinary-operator handoff and loader/SDK outcome. Rejection remains a
failed qualification, not a successful launch. If native packaging requires a
valid new signature, a correctly disclosed governed derived-package signing
workflow may be evaluated; no universal prohibition on legitimate signing is
inferred. No re-signing or packaging-policy change was performed by this
followup, and no new human approval gate is invented.

Private reproducibility evidence:
`.vrooli/work/openrgb-mac-signature-artifact-facts.json` and
`openrgb-mac-signature-primary-references.json`. The latter records Apple source
blob identities as corroboration, not the identity of a target OS build.
