package main

// --- Test helpers ---

// AddStagedFile adds a file to the staged state.
func (f *FakeGitRunner) AddStagedFile(path string) *FakeGitRunner {
	f.Staged[path] = "+staged content"
	return f
}

// AddUnstagedFile adds a file to the unstaged (modified) state.
func (f *FakeGitRunner) AddUnstagedFile(path string) *FakeGitRunner {
	f.Unstaged[path] = "+modified content"
	return f
}

// AddUntrackedFile adds an untracked file.
func (f *FakeGitRunner) AddUntrackedFile(path string) *FakeGitRunner {
	f.Untracked = append(f.Untracked, path)
	return f
}

// AddConflictFile adds a file with merge conflicts.
func (f *FakeGitRunner) AddConflictFile(path string) *FakeGitRunner {
	f.Conflicts = append(f.Conflicts, path)
	return f
}

// WithBranch sets the branch state.
func (f *FakeGitRunner) WithBranch(head, upstream string, ahead, behind int) *FakeGitRunner {
	f.Branch.Head = head
	f.Branch.Upstream = upstream
	f.Branch.Ahead = ahead
	f.Branch.Behind = behind
	if _, exists := f.LocalBranches[head]; !exists {
		f.LocalBranches[head] = FakeBranchRef{
			Name:         head,
			Upstream:     upstream,
			OID:          f.Branch.OID,
			LastCommitAt: "2025-01-01 00:00:00 +0000",
		}
	}
	return f
}

func (f *FakeGitRunner) WithLocalBranch(name, upstream string, oid string) *FakeGitRunner {
	if oid == "" {
		oid = "abc123def456"
	}
	f.LocalBranches[name] = FakeBranchRef{
		Name:         name,
		Upstream:     upstream,
		OID:          oid,
		LastCommitAt: "2025-01-01 00:00:00 +0000",
	}
	return f
}

func (f *FakeGitRunner) WithRemoteBranch(name string, oid string) *FakeGitRunner {
	if oid == "" {
		oid = "abc123def456"
	}
	f.RemoteBranches[name] = FakeBranchRef{
		Name:         name,
		OID:          oid,
		LastCommitAt: "2025-01-01 00:00:00 +0000",
	}
	return f
}

// WithFileFrequency sets the simulated file frequency data.
func (f *FakeGitRunner) WithFileFrequency(freq map[string]int) *FakeGitRunner {
	f.FileFrequency = freq
	return f
}

// WithNotARepository simulates a non-git directory.
func (f *FakeGitRunner) WithNotARepository() *FakeGitRunner {
	f.IsRepository = false
	return f
}

// WithGitUnavailable simulates git not being installed.
func (f *FakeGitRunner) WithGitUnavailable() *FakeGitRunner {
	f.GitAvailable = false
	return f
}

// WithRepoRoot sets the repository root path.
func (f *FakeGitRunner) WithRepoRoot(root string) *FakeGitRunner {
	f.RepoRoot = root
	return f
}

// WithRemoteURL sets the remote URL.
func (f *FakeGitRunner) WithRemoteURL(url string) *FakeGitRunner {
	f.RemoteURL = url
	return f
}

// AssertCalled verifies a method was called.
func (f *FakeGitRunner) AssertCalled(method string) bool {
	for _, call := range f.Calls {
		if call.Method == method {
			return true
		}
	}
	return false
}

// AssertNotCalled verifies a method was not called.
func (f *FakeGitRunner) AssertNotCalled(method string) bool {
	return !f.AssertCalled(method)
}

// AssertCalledWith verifies a method was called with the given arg present in its recorded args.
func (f *FakeGitRunner) AssertCalledWith(method string, arg string) bool {
	for _, call := range f.Calls {
		if call.Method != method {
			continue
		}
		for _, a := range call.Args {
			if a == arg {
				return true
			}
		}
	}
	return false
}

// CallCount returns the number of times a method was called.
func (f *FakeGitRunner) CallCount(method string) int {
	count := 0
	for _, call := range f.Calls {
		if call.Method == method {
			count++
		}
	}
	return count
}
