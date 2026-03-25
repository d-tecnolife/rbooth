# rbooth

A mobile website built with Golang and JavaScript that allows users to upload photos to a photoboard with live updates to see a collection of everyone's photos.

Configuration:

- Set `APP_NAME`, `APP_BASE_URL`, `ADMIN_PASSWORD`, and `AUTH_SECRET` in `.env` for Docker deployments.
- Set `DISPLAY_TIMEZONE` in `.env` if you want timestamps rendered in a specific timezone; the default is `UTC`.
- Optional personalization envs are `APP_SUBTITLE`, `HOME_WELCOME_TITLE`, `HOME_WELCOME_BODY`, `HOME_ACCESS_TITLE`, `HOME_ACCESS_BODY`, `BOARD_EMPTY_TITLE`, `BOARD_EMPTY_BODY`, `ADMIN_INTRO_TITLE`, and `ADMIN_INTRO_BODY`.
- Optional envs are `PORT` and `DATA_DIR`; defaults are `8080` and `data`.

Docker deployment:

- Build the container locally with `docker build -t rbooth .`.
- For server deployment, copy `.env.example` to `.env` and set `APP_NAME`, `APP_BASE_URL`, `ADMIN_PASSWORD`, and `AUTH_SECRET`.
- Personalize the event copy in `.env` if you want to change the subtitle, home welcome text, board empty-state text, or admin intro without rebuilding the image.
- `docker-compose.yml` mounts persistent app state at `./data` and maps the host media volume `/mnt/storage/media/rbooth` to `/app/media` in the container.
- The compose service only binds to `127.0.0.1:${HOST_PORT}` so a local reverse proxy or tunnel can forward traffic to it.
- The image is designed to run with env vars for secrets; do not depend on `config.json` in the container image.

Editor assets:

- Drop custom background images into `web/static/editor-assets/backdrops/`.
- Drop custom frame overlay images into `web/static/editor-assets/frames/`.
- Supported file types are `.png`, `.jpg`, `.jpeg`, `.webp`, and `.gif`.
- Reload the capture page and any new file will appear as a new option automatically.
- Backdrop assets are stretched to the full canvas.
- Frame assets are drawn as full-canvas overlays, so transparent PNGs work best.

Recommended free event deployment:

- Update `.env` so `APP_BASE_URL` matches your deployed URL.
- Restart the app with `docker compose up -d` after changing `APP_BASE_URL` so QR generation uses the URL.
- Open `/admin`, log in, and use the QR code there for guest access during the event.

CI/CD:

- GitHub Actions runs `go test ./...` on every push and pull request via `.github/workflows/ci.yml`.
- `.github/workflows/docker-publish.yml` builds and publishes the Docker image to `ghcr.io/d-tecnolife/rbooth`.
- Pushes to the default branch publish `:latest` and `:sha-*` tags; other branches also publish branch-tagged images.

Usage:

- Scan a QR code that links to the site
- On the site interface, take a picture
- View the picture before uploading, adding filters/changing the background
- Upload the picture onto a board displaying everyone else's picture taken with realtime updates

Implementation:

- Picture taken will be temporarily cached on a server for image manipulation with local tool calls
- Store photos on the locally mounted media volume after editing/confirmation
- Application pulls photos from the mounted storage to display on the board

Scope:

- Maybe around 10 people will be using the site at a time concurrently max
- Will not be for business use, so HA not strictly required
- Cache server not required either
