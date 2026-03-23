# rbooth User Requirements

## Product Requirements

- The project is named `rbooth`.
- Users should be able to upload photos to a site.
- The site should show live updates so users can see everyone's photos in real time.
- Users should be able to scan a QR code that links to the site.
- The site should allow a user to take a picture from the interface.
- The site should allow a user to view the picture before uploading.
- The site should support adding filters and changing the background before upload.
- Uploaded pictures should appear on a board displaying everyone else's pictures.

## Implementation Direction

- Pictures may be temporarily cached on a server for image manipulation.
- The project should store photos in cloud storage after editing and confirmation.
- The project should be designed with Azure as the preferred cloud provider.

## Scope Constraints

- Expected peak concurrency is around 10 users at a time.
- The project is not intended for business use.
- High availability is not strictly required.
- A cache server is not strictly required.

## Delivery Requirements

- The application should be browser-based so people do not have to download an app.
- The website should be created in this repository.
- The root directory of the site should be the photo board itself.
- The board should visually mimic a real photo board with slight angle changes per photo, as if the photos were attached to a physical board.
- The colour scheme should use girly pastel pink styling.
