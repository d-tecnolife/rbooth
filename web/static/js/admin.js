const adminPhotoCount = document.getElementById("adminPhotoCount");
const cardsGrid = document.querySelector(".gallery-strip .grid.cards");

if (cardsGrid) {
  function updateAdminCount() {
    const total = cardsGrid.querySelectorAll(".thumb-card").length;
    if (adminPhotoCount) {
      adminPhotoCount.textContent = String(total);
    }
  }

  function ensureEmptyState() {
    if (cardsGrid.querySelector(".thumb-card")) {
      return;
    }
    if (document.getElementById("adminEmptyState")) {
      return;
    }

    const empty = document.createElement("article");
    empty.id = "adminEmptyState";
    empty.className = "card empty";
    empty.innerHTML = "<p>No photos yet. Open the camera page on a phone and publish one.</p>";
    cardsGrid.append(empty);
  }

  async function deletePhoto(id, button) {
    button.disabled = true;
    button.textContent = `Deleting ${id}...`;

    try {
      const response = await fetch(`/api/photos/${encodeURIComponent(id)}`, { method: "DELETE" });
      if (!response.ok) {
        throw new Error(`Delete failed with status ${response.status}`);
      }

      const card = cardsGrid.querySelector(`[data-photo-id="${CSS.escape(id)}"]`);
      if (card) {
        card.remove();
      }
      ensureEmptyState();
      updateAdminCount();
    } catch (error) {
      console.error(error);
      button.disabled = false;
      button.textContent = `Delete ${id}`;
      window.alert(`Failed to delete photo ${id}.`);
    }
  }

  cardsGrid.addEventListener("click", (event) => {
    const button = event.target.closest("[data-delete-photo]");
    if (!button) {
      return;
    }

    const id = button.dataset.deletePhoto;
    if (!id) {
      return;
    }
    if (!window.confirm(`Delete photo #${id}?`)) {
      return;
    }

    const empty = document.getElementById("adminEmptyState");
    if (empty) {
      empty.remove();
    }
    deletePhoto(id, button);
  });

  updateAdminCount();
}
