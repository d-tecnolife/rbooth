# rbooth

A mobile website built with Golang that allows users to upload photos to a photoboard with live updates to see a collection of everyone's photos.

Usage:

- Scan a QR code that links to the site
- On the site interface, take a picture
- View the picture before uploading, adding filters/changing the background
- Upload the picture onto a board displaying everyone else's picture taken with realtime updates

Implementation:

- Picture taken will be temporarily cached on a server for image manipulation with either local or cloud tool calls
- Store photos in cloud storage (Google Drive, Azure, AWS) after editing/confirmation
- Application pulls photos from the storage to display on the board

Scope:

- Maybe around 10 people will be using the site at a time concurrently max
- Will not be for business use, so HA not strictly required
- Cache server not required either
