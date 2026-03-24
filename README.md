# rbooth

A mobile website built with Golang and JavaScript that allows users to upload photos to a photoboard with live updates to see a collection of everyone's photos.

Configuration:

- Set `APP_BASE_URL`, `ADMIN_PASSWORD`, and `AUTH_SECRET` in `.env` for Docker deployments.
- `MEDIA_DIR` may be changed to your media storage location.
- Optional envs are `PORT` and `DATA_DIR`; defaults are `8080` and `data`.

Docker deployment:

- Build the container locally with `docker build -t rbooth .`.
- For server deployment, copy `.env.example` to `.env` and set `APP_BASE_URL`, `ADMIN_PASSWORD`, and `AUTH_SECRET`.
- `docker-compose.yml` mounts persistent app state at `./data` and the media volume at `/mnt/storage/media/rbooth`.
- The compose service only binds to `127.0.0.1:${HOST_PORT}` so a local reverse proxy or tunnel can forward traffic to it.
- The image is designed to run with env vars for secrets; do not depend on `config.json` in the container image.

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
