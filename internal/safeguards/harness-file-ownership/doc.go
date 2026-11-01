// Package harnessfileownership hands root-owned coding-agent files back to the
// operator, which is the repair behind the update state "needs setup".
//
// An elevated run can leave a root-owned regular file at ~/.local/bin/agy. The
// user can write the directory, so nothing looks wrong until an update has to
// replace the file. Web Console never elevates and only shows
// `sudo vrooli setup --include-optional`; this safeguard is what makes that
// command true. It changes ownership and nothing else: moving the file aside and
// linking Vrooli's copy stays a user-privileged, reversible action in
// `vrooli resource update --action link-managed-copy`.
package harnessfileownership
