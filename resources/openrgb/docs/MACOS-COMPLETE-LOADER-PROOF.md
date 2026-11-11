# macOS complete loader proof

The original release bundles are retained as source artifacts. The headless
serving profiles verify those complete originals, then omit the exact orphan
`Contents/Frameworks/QtCore5Compat.framework` subtree. No remaining executable,
library or page-signature bytes change. Original-source adoption still requires
the complete original bundle and preserves it.

The orphan imports three ICU libraries and QtCore through absolute Homebrew
paths. Neither architecture has an incoming declared dependency on this
framework. Accepting its executable hash while ignoring these imports was an
incomplete loader proof. The derived profiles keep all other 77 native images
and have separate source, omitted-subtree, recipe and serving identities.

| Identity | Intel | Apple Silicon |
| --- | --- | --- |
| Serving tree SHA256 | `61ead4311d86af4c89fdbfd2868b474dae3a2242216b6a506bd295336f2e2519` | `b777a233d74db1250f2a100571bb3880fa8975cc6eca6ee71fc44efe3f2cdab5` |
| Canonical recipe SHA256 | `8b03da61b917f96d22c2cbc9c35ba8921f0cdb0303be17cf38dd294c66300ec0` | `879c9b9a299a8096d7dd968986ab8ca2e31ef4e98ae40ab94991d2f3d385ac81` |

The declared recipes are `config/darwin-amd64-headless-serving-recipe.json` and
`config/darwin-arm64-headless-serving-recipe.json`. Their canonical JSON agrees
with the compiled profile. The recipe participates in the stage address and
the sealed selected-source metadata, preventing an old original stage from
being reinterpreted as a derived one.

## Loader metadata and path authority

`platform.ReadMachOLoaderMetadata` reads the held image and its explicit extent.
It covers normal, weak, reexport, upward and lazy dylib commands, alternate
`dylib_use_command` flags, the dynamic linker, install identity, runpaths and
embedded dyld environment. Unknown forms and malformed command/string extents
refuse qualification. `debug/macho.ImportedLibraries` is not a complete loader
inventory and is no longer used by this native resource gate.

The resource applies its own vendor policy rather than the maintained control
plane's system-only policy. Each image must match the pinned native CPU and
image format. The executable uses `/usr/lib/dyld`; custom interpreters and
embedded dyld environment are refused. Every declared dependency must resolve
to a native image inside the immutable pinned bundle or a canonical Apple
system image. Every system dependency, including weak forms, participates in
the active dyld-cache prerequisite check; the two current serving profiles each
declare 32 unique system images.

For private imports, `@loader_path` and `@executable_path` resolve within the
bundle through validated in-tree framework links. An `@rpath` dependency must
have a usable self/executable runpath, and all other private runpath candidates
must resolve to that same canonical image. This conservative agreement makes
private search order irrelevant. The graph models runpath inheritance conservatively: private candidates across every possible ancestor agree, while a foreign runpath must not reach any runpath-dependent descendant. This includes cycles. Thus ordering within any accepted dependency-chain stack cannot select another private target or activate a foreign candidate. This is a proof for the fixed deployed
bundle, not a general dyld emulator.

Twenty-two Qt plugins retain `@loader_path/../../../../lib`. In this deployment
it resolves to exactly the adjacent immutable stage's `lib` directory, not the
operator's mutable runtime directory. That location must be absent. The stage
root admits only the bundle, retained source and serving metadata; adding an
adjacent provider fails recovery and serving verification. Native immutable
ancestry prevents an ordinary operator from creating that provider after the
check.

Three C libraries retain absolute build-time Cellar runpaths: libdbus, libjasper
and libjpeg. Only the six exact architecture-specific image hashes and runpath
strings are admitted. Offline inspection of each exact image finds no dynamic
loader, Qt or Objective-C loader imports, no loader-name or `@rpath` bytes and
no constructor sections/commands. Their complete declared descendant graphs
contain no runpath-relative requests. Maintained regressions reproduce these
properties from the official bytes. These bounded facts establish that the
build-time paths do not participate in this declared loader graph; they do not
establish confinement against arbitrary code or arbitrary future `dlopen`
behavior. A new image or path requires new evidence and is rejected meanwhile.

Apple documents runpath deployment in [Run-Path Dependent Libraries](https://developer.apple.com/library/archive/documentation/DeveloperTools/Conceptual/DynamicLibraries/100-Articles/RunpathDependentLibraries.html).
The raw command layouts are owned by the shared platform parser using Apple's
`mach-o/loader.h`; the resource does not maintain a second binary decoder.

## Signature and execution limits

Omitting the orphan changes the Apple Silicon app's resource envelope: its
original `Contents/_CodeSignature/CodeResources` references the omitted
framework. The Intel archive contains no root CodeResources file. The
profile does **not** claim an unchanged upstream whole-app resource signature,
notarization or Gatekeeper assessment. Remaining Mach-O page-signature bytes
are unchanged; no binary patch, ad hoc signature bypass or new signing step is
performed. Actual direct execution, AMFI/Gatekeeper behavior, native ACL/lock
enforcement, headless constructors, HID access and SDK operation remain target
qualification obligations for this explicitly derived local serving bundle.

[The bounded signature followup](MACOS-SIGNATURE-ARTIFACT-FACTS.md) records the
actual embedded signature/resource records, the required missing ARM nested
framework, and the real direct-exec caller. Neither offline page hashes nor
that caller establishes native object-level or whole-app validation. Legitimate
disclosed signing remains an in-scope packaging option if target requirements
make it necessary; this followup performs no signing or policy bypass.

The source and offline fixtures cover both original/derived trees, complete
metadata for 156 original native images, both 77-image serving graphs, all five
dependency command forms, missing cache providers, foreign/unresolved loads,
custom dyld, embedded environment, malformed commands, inherited/cyclic runpath
flows, competing private candidates, managed/adopted staging, unknown publish
acknowledgement, ordinary recovery and adjacent-provider refusal. These do not
replace native runtime or physical-light evidence.
