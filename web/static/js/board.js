const boardShell = document.querySelector(".shell-board");

if (boardShell) {
  const boardGrid = document.getElementById("boardGrid");
  const boardCount = document.getElementById("boardCount");

  function updateBoardCount() {
    const total = boardGrid.querySelectorAll(".photo-card:not(.photo-card-empty)").length;
    if (boardCount) {
      boardCount.textContent = `${total} photos pinned on the board`;
    }
  }

  function createCard(photo) {
    const card = document.createElement("article");
    card.className = "photo-card";
    card.dataset.photoId = photo.id;
    card.innerHTML = `
      <img src="${photo.display_url}" alt="${photo.caption || ""}">
      <div class="photo-meta">
        <strong>${photo.caption || ""}</strong>
        <span>${new Date(photo.created_at).toLocaleTimeString()}</span>
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

    const card = document.createElement("article");
    card.className = "photo-card photo-card-empty";
    card.innerHTML = `
      <div class="photo-meta">
        <strong>The board is ready</strong>
        <span>New snapshots will pin themselves here and stack into the collage live.</span>
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
