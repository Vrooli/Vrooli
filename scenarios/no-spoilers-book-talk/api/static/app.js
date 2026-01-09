const state = {
  userId: localStorage.getItem("bookTalkUserId") || "demo-reader",
  books: [],
  selectedBookId: null,
};

const dom = {
  status: document.getElementById("status"),
  userForm: document.getElementById("user-form"),
  userId: document.getElementById("user-id"),
  uploadForm: document.getElementById("upload-form"),
  refreshBooks: document.getElementById("refresh-books"),
  booksList: document.getElementById("books-list"),
  booksEmpty: document.getElementById("books-empty"),
  progressCard: document.getElementById("progress-card"),
  progressForm: document.getElementById("progress-form"),
  progressCurrent: document.getElementById("progress-current"),
  progressMinutes: document.getElementById("progress-minutes"),
  progressNotes: document.getElementById("progress-notes"),
  progressSummary: document.getElementById("progress-summary"),
  chatCard: document.getElementById("chat-card"),
  chatForm: document.getElementById("chat-form"),
  chatMessage: document.getElementById("chat-message"),
  chatPosition: document.getElementById("chat-position"),
  chatTemperature: document.getElementById("chat-temperature"),
  chatHistory: document.getElementById("chat-history"),
};

dom.userId.value = state.userId;

function setStatus(message, tone = "info") {
  if (!message) {
    dom.status.style.display = "none";
    dom.status.textContent = "";
    dom.status.removeAttribute("data-tone");
    return;
  }
  dom.status.textContent = message;
  dom.status.setAttribute("data-tone", tone);
  dom.status.style.display = "block";
}

function formatDate(value) {
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(value));
  } catch (err) {
    return value;
  }
}

function formatPercent(value) {
  if (typeof value !== "number" || Number.isNaN(value)) {
    return "0%";
  }
  return `${Math.min(100, Math.max(0, value)).toFixed(1)}%`;
}

async function request(url, options = {}) {
  const response = await fetch(url, options);
  let payload = null;

  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    payload = await response.json().catch(() => null);
  } else {
    const text = await response.text();
    payload = text ? { message: text } : null;
  }

  if (!response.ok) {
    const errorMessage =
      (payload && (payload.error || payload.message)) ||
      `Request failed (${response.status})`;
    throw new Error(errorMessage);
  }

  return payload;
}

function renderBooks(books) {
  dom.booksList.innerHTML = "";
  if (!books.length) {
    dom.booksEmpty.classList.remove("hidden");
    return;
  }

  dom.booksEmpty.classList.add("hidden");

  books.forEach((book) => {
    const item = document.createElement("li");
    item.className = "book-item";
    item.dataset.bookId = book.id;

    if (state.selectedBookId === book.id) {
      item.classList.add("active");
    }

    const title = book.title || "Untitled Manuscript";
    const status = (book.processing_status || "unknown").toUpperCase();
    const words = book.total_words ? `${book.total_words.toLocaleString()} words` : "Awaiting analysis";
    const uploaded = book.created_at ? formatDate(book.created_at) : "";
    let progressMarkup = "<em>No reader progress yet</em>";

    if (book.user_progress) {
      const percent = formatPercent(book.user_progress.percentage_complete);
      progressMarkup = `${percent} complete · current chunk ${book.user_progress.current_position}`;
    }

    item.innerHTML = `
      <h3>${title}</h3>
      <div class="book-meta">Status: ${status}</div>
      <div class="book-meta">${words}${uploaded ? ` · Uploaded ${uploaded}` : ""}</div>
      <div class="book-meta">${progressMarkup}</div>
      <div class="book-actions">
        <button type="button" class="open-book">Open reader tools</button>
      </div>
    `;

    item
      .querySelector(".open-book")
      .addEventListener("click", () => selectBook(book.id));

    dom.booksList.appendChild(item);
  });
}

function markActiveBook() {
  Array.from(dom.booksList.children).forEach((item) => {
    if (!(item instanceof HTMLElement)) return;
    if (item.dataset.bookId === state.selectedBookId) {
      item.classList.add("active");
    } else {
      item.classList.remove("active");
    }
  });
}

async function fetchBooks(showStatus = false) {
  if (!state.userId) {
    renderBooks(state.books);
    return;
  }

  if (showStatus) {
    setStatus("Loading library…");
  }

  try {
    const data = await request(`/api/v1/books?user_id=${encodeURIComponent(state.userId)}`);
    state.books = Array.isArray(data) ? data : [];
    renderBooks(state.books);
    markActiveBook();
    if (showStatus) {
      setStatus(`Loaded ${state.books.length} book${state.books.length === 1 ? "" : "s"}.`, "success");
    } else {
      setStatus("");
    }
  } catch (error) {
    setStatus(error.message, "danger");
  }
}

async function selectBook(bookId) {
  state.selectedBookId = bookId;
  markActiveBook();
  dom.progressCard.classList.add("hidden");
  dom.chatCard.classList.add("hidden");
  dom.chatHistory.innerHTML = "";
  dom.progressSummary.textContent = "";

  if (!bookId || !state.userId) {
    return;
  }

  setStatus("Loading book tools…");

  try {
    const detail = await request(`/api/v1/books/${bookId}?user_id=${encodeURIComponent(state.userId)}`);
    const { book, user_progress: progress } = detail;
    updateProgressUI(book, progress || null);
    await fetchConversations(bookId);
    setStatus(`Ready to continue reading “${book.title || "your book"}”.`, "success");
  } catch (error) {
    setStatus(error.message, "danger");
  }
}

function updateProgressUI(book, progress) {
  const isReadyForChat = book.processing_status === "completed";

  dom.progressCard.classList.remove("hidden");
  dom.chatCard.classList.toggle("hidden", !isReadyForChat);

  const currentChunk = progress ? progress.current_position : 0;
  dom.progressCurrent.value = currentChunk;
  dom.chatPosition.value = currentChunk;
  dom.progressNotes.value = progress ? progress.notes || "" : "";
  dom.progressMinutes.value = 15;

  const percent = progress ? formatPercent(progress.percentage_complete) : "0%";
  dom.progressSummary.textContent = `You have unlocked ${percent} of ${book.title || "this book"}. Total chunks: ${book.total_chunks ?? "unknown"}.`;

  if (!isReadyForChat) {
    dom.chatHistory.innerHTML = `<div class="empty">We are still processing this book. Once ready, you can start a spoiler-safe chat.</div>`;
  }
}

async function fetchConversations(bookId) {
  if (!state.userId) return;
  try {
    const data = await request(
      `/api/v1/books/${bookId}/conversations?user_id=${encodeURIComponent(state.userId)}`
    );
    renderConversations(data?.conversations || []);
  } catch (error) {
    dom.chatHistory.innerHTML = `<div class="empty">${error.message}</div>`;
  }
}

function renderConversations(conversations) {
  if (!conversations.length) {
    dom.chatHistory.innerHTML = `<div class="empty">Ask your first question to begin a spoiler-aware conversation.</div>`;
    return;
  }

  dom.chatHistory.innerHTML = "";

  conversations.forEach((conversation) => {
    const container = document.createElement("article");
    container.className = "message";

    const created = formatDate(conversation.created_at);
    const sources = (conversation.sources_referenced || []).join(", ");

    container.innerHTML = `
      <h3>You asked</h3>
      <p>${conversation.user_message || "(message missing)"}</p>
      <div class="meta">Chunk ${conversation.user_position} · ${created}</div>
      <h3>AI replied</h3>
      <p>${conversation.ai_response || "(response missing)"}</p>
      <div class="meta">Sources: ${sources || "Context chunks"}</div>
    `;

    dom.chatHistory.appendChild(container);
  });
}

function ensureUserId() {
  if (state.userId) {
    return true;
  }
  setStatus("Please pick a reader profile before continuing.", "danger");
  dom.userId.focus();
  return false;
}

// Event bindings

dom.userForm.addEventListener("submit", (event) => {
  event.preventDefault();
  const value = dom.userId.value.trim();
  if (!value) {
    setStatus("Reader ID is required.", "danger");
    dom.userId.focus();
    return;
  }
  state.userId = value;
  localStorage.setItem("bookTalkUserId", value);
  setStatus(`Using reader profile “${value}”.`, "success");
  fetchBooks();
  if (state.selectedBookId) {
    selectBook(state.selectedBookId);
  }
});

dom.refreshBooks.addEventListener("click", () => {
  fetchBooks(true);
});

dom.uploadForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!ensureUserId()) return;

  const formData = new FormData(dom.uploadForm);
  formData.append("user_id", state.userId);

  setStatus("Uploading book…");
  dom.uploadForm.querySelector("button[type=submit]").disabled = true;

  try {
    await request("/api/v1/books/upload", {
      method: "POST",
      body: formData,
    });

    setStatus("Book uploaded successfully! Processing will continue in the background.", "success");
    dom.uploadForm.reset();
    fetchBooks();
  } catch (error) {
    setStatus(error.message, "danger");
  } finally {
    dom.uploadForm.querySelector("button[type=submit]").disabled = false;
  }
});

dom.progressForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!ensureUserId() || !state.selectedBookId) return;

  const current = Number(dom.progressCurrent.value || 0);
  const minutes = Number(dom.progressMinutes.value || 0);
  const notes = dom.progressNotes.value.trim();

  try {
    await request(`/api/v1/books/${state.selectedBookId}/progress`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        user_id: state.userId,
        current_position: current,
        position_type: "chunk",
        position_value: current,
        notes,
        reading_time_minutes: minutes,
      }),
    });
    setStatus("Progress updated. Keep reading!", "success");
    fetchBooks();
    selectBook(state.selectedBookId);
  } catch (error) {
    setStatus(error.message, "danger");
  }
});

dom.chatForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (!ensureUserId() || !state.selectedBookId) return;

  const message = dom.chatMessage.value.trim();
  const position = Number(dom.chatPosition.value || 0);
  const temperature = Number(dom.chatTemperature.value || 0.7);

  if (!message) {
    setStatus("Ask a question to start the conversation.", "danger");
    dom.chatMessage.focus();
    return;
  }

  try {
    await request(`/api/v1/books/${state.selectedBookId}/chat`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        message,
        user_id: state.userId,
        current_position: position,
        position_type: "chunk",
        temperature,
      }),
    });

    dom.chatMessage.value = "";
    setStatus("Response generated based on your safe reading zone.", "success");
    await fetchConversations(state.selectedBookId);
  } catch (error) {
    setStatus(error.message, "danger");
  }
});

// Initial load
fetchBooks();
if (state.selectedBookId) {
  selectBook(state.selectedBookId);
}
