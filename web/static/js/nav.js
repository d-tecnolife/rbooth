const navToggle = document.querySelector("[data-nav-open]");
const navClose = document.querySelector("[data-nav-close]");
const menuScrim = document.querySelector("[data-nav-scrim]");

function setMenuState(open) {
  document.body.classList.toggle("menu-open", open);
}

navToggle?.addEventListener("click", () => setMenuState(true));
navClose?.addEventListener("click", () => setMenuState(false));
menuScrim?.addEventListener("click", () => setMenuState(false));

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    setMenuState(false);
  }
});
