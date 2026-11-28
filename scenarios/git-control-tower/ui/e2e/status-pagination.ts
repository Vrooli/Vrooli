import { fetchRepoStatus } from "../src/lib/api";

const state = document.querySelector<HTMLElement>("#status-state")!;
const resultElement = document.querySelector<HTMLElement>("#status-result")!;
const refresh = document.querySelector<HTMLButtonElement>("#refresh-status")!;
const repoId = new URLSearchParams(window.location.search).get("repoId") ?? "1";

async function loadStatus(): Promise<void> {
  state.dataset.state = "loading";
  state.textContent = "Loading complete status…";
  resultElement.textContent = "";
  try {
    const status = await fetchRepoStatus(repoId);
    const files = status.files;
    const counts = {
      staged: files.staged.length,
      unstaged: files.unstaged.length,
      untracked: files.untracked.length,
      conflicts: files.conflicts.length,
      ignored: files.ignored.length,
    };
    state.dataset.state = "complete";
    state.textContent = `Complete: ${Object.values(counts).reduce((sum, value) => sum + value, 0)} status entries`;
    resultElement.textContent = JSON.stringify({ counts, files: { staged: files.staged, unstaged: files.unstaged, untracked: files.untracked, conflicts: files.conflicts, ignored: files.ignored } });
  } catch (error) {
    state.dataset.state = "error";
    state.textContent = error instanceof Error ? error.message : String(error);
  }
}

refresh.addEventListener("click", () => void loadStatus());
void loadStatus();
