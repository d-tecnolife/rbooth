const boardShell = document.querySelector(".shell-board");

if (boardShell) {
  const boardGrid = document.getElementById("boardGrid");

  function createCard(photo) {
    const card = document.createElement("article");
    card.className = "photo-card";
    card.innerHTML = `
      <img src="${photo.display_url}" alt="${photo.caption || ""}">
      <div class="photo-meta">
        <strong>${photo.caption || "Fresh upload"}</strong>
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
}
