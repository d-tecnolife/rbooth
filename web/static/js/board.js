const boardShell = document.querySelector(".shell-board");

if (boardShell) {
  const boardGrid = document.getElementById("boardGrid");
  const boardCount = document.getElementById("boardCount");
  const displayTimeZone = boardGrid.dataset.timezone || "UTC";
  const boardTimeFormatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: displayTimeZone,
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
    timeZoneName: "short",
  });

  function updateBoardCount() {
    const total = boardGrid.querySelectorAll(".photo-card:not(.photo-card-empty)").length;
    if (boardCount) {
      boardCount.textContent = `${total} photos pinned on the wall`;
    }
  }

  function createCard(photo) {
    const card = document.createElement("article");
    card.className = "photo-card";
    card.dataset.photoId = photo.id;
    card.innerHTML = `
      <img src="${photo.display_url}" alt="${photo.caption || ""}">
      <div class="photo-meta">
        <strong class="photo-caption">${photo.caption || ""}</strong>
        <span>${boardTimeFormatter.format(new Date(photo.created_at))}</span>
      </div>
    `;
    return card;
  }

  function insertPhoto(photo) {
    const empty = boardGrid.querySelector(".photo-card-empty");
    if (empty) {
      empty.remove();
    }
    boardGrid.prepend(createCard(photo));
    updateBoardCount();
  }

  function ensureEmptyState() {
    if (boardGrid.querySelector(".photo-card")) {
      return;
    }
    const emptyTitle = boardGrid.dataset.emptyTitle || "the wall is ready";
    const emptyBody =
      boardGrid.dataset.emptyBody ||
      "new snapshots will pin themselves here and stack into the collage live.";

    const card = document.createElement("article");
    card.className = "photo-card photo-card-empty";
    card.innerHTML = `
      <div class="photo-meta">
        <strong>${emptyTitle}</strong>
        <span>${emptyBody}</span>
      </div>
    `;
    boardGrid.append(card);
  }

  function removePhoto(id) {
    const card = boardGrid.querySelector(`[data-photo-id="${CSS.escape(id)}"]`);
    if (!card) {
      return;
    }
    card.remove();
    ensureEmptyState();
    updateBoardCount();
  }

  const stream = new EventSource("/stream");
  stream.addEventListener("photo", (event) => {
    try {
      const payload = JSON.parse(event.data);
      insertPhoto(payload.photo);
    } catch (error) {
      console.error(error);
    }
  });

  stream.addEventListener("photo-delete", (event) => {
    try {
      const payload = JSON.parse(event.data);
      if (payload.id) {
        removePhoto(payload.id);
      }
    } catch (error) {
      console.error(error);
    }
  });

  updateBoardCount();
}
